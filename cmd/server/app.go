package main

import (
	"database/sql"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "modernc.org/sqlite"

	"rustymanager/internal/db"
	"rustymanager/internal/handler"
	authmw "rustymanager/internal/middleware"
	"rustymanager/internal/store"
	"rustymanager/web"
)

type renderer struct {
	fsys fs.FS
	base *template.Template
}

func newRenderer(fsys fs.FS) (*renderer, error) {
	base, err := template.ParseFS(fsys, "templates/layout.html")
	if err != nil {
		return nil, err
	}
	return &renderer{fsys: fsys, base: base}, nil
}

func (r *renderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	t, err := r.base.Clone()
	if err != nil {
		return err
	}
	if _, err = t.ParseFS(r.fsys, "templates/"+name); err != nil {
		return err
	}
	return t.ExecuteTemplate(w, "layout", data)
}

func newApp(dsn string) (*echo.Echo, error) {
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := store.Migrate(database); err != nil {
		return nil, err
	}

	queries := db.New(database)
	s := store.New(queries)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	staticSub, err := fs.Sub(web.FS, "static")
	if err != nil {
		log.Fatalf("static subfs: %v", err)
	}
	e.StaticFS("/static", staticSub)

	rend, err := newRenderer(web.FS)
	if err != nil {
		return nil, err
	}
	e.Renderer = rend

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	a := handler.NewAuth()
	e.GET("/login", a.LoginPage)
	e.POST("/login", a.Login)
	e.POST("/logout", a.Logout)

	p := e.Group("")
	p.Use(authmw.RequireAuth)

	h := handler.NewProjects(s)
	p.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/projects")
	})
	p.GET("/projects", h.Index)
	p.GET("/projects/new", h.New)
	p.POST("/projects", h.Create)
	p.GET("/projects/:id", h.Show)
	p.GET("/projects/:id/edit", h.Edit)
	p.POST("/projects/:id", h.Update)
	p.POST("/projects/:id/delete", h.Delete)

	u := handler.NewUsers(s)
	p.GET("/users", u.Index)
	p.GET("/users/new", u.New)
	p.POST("/users", u.Create)
	p.GET("/users/:id/edit", u.Edit)
	p.POST("/users/:id", u.Update)
	p.POST("/users/:id/delete", u.Delete)

	k := handler.NewKanban(s)
	p.GET("/projects/:id/kanban/new", k.New)
	p.POST("/projects/:id/kanban", k.Create)
	p.POST("/projects/:id/kanban/:itemID/status", k.UpdateStatus)
	p.POST("/projects/:id/kanban/:itemID/delete", k.Delete)
	p.POST("/projects/:id/kanban/done/delete-all", k.DeleteAllDone)

	chat := handler.NewChatChannel(queries)
	p.GET("/projects/:id/ws", chat.HandleWS)

	commits := handler.NewCommits(s)
	p.GET("/projects/:id/commits", commits.List)

	return e, nil
}
