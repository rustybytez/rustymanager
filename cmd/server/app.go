package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "modernc.org/sqlite"

	"rustymanager/internal/db"
	"rustymanager/internal/handler"
	authmw "rustymanager/internal/middleware"
	"rustymanager/internal/push"
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
	if m, ok := data.(map[string]any); ok {
		if user := c.Get(authmw.CurrentUserKey); user != nil {
			m["CurrentUser"] = user
		}
		if project := c.Get(authmw.CurrentProjectKey); project != nil {
			m["CurrentProject"] = project
		}
	}
	return t.ExecuteTemplate(w, "layout", data)
}

func loadVAPIDKeys() (pubKey, privKey string) {
	return os.Getenv("VAPID_PUBLIC_KEY"), os.Getenv("VAPID_PRIVATE_KEY")
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

	// Service workers must be served from the root path to control all pages.
	e.GET("/sw.js", func(c echo.Context) error {
		f, err := web.FS.Open("static/sw.js")
		if err != nil {
			return echo.ErrNotFound
		}
		defer f.Close()
		c.Response().Header().Set("Content-Type", "application/javascript")
		c.Response().Header().Set("Service-Worker-Allowed", "/")
		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		_, err = io.Copy(c.Response().Writer, f)
		return err
	})

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

	// Auth-protected group (no user selection required)
	p := e.Group("")
	p.Use(authmw.RequireAuth)
	p.POST("/logout", a.Logout)

	u := handler.NewUsers(s)
	p.GET("/select-user", u.SelectPage)
	p.POST("/select-user", u.Select)
	p.POST("/switch-user", u.SwitchUser)

	// Auth + user-selection required
	r := p.Group("")
	r.Use(authmw.RequireUser(s))
	r.Use(authmw.LoadProject(s))

	vapidPub, vapidPriv := loadVAPIDKeys()
	vapidSubscriber := os.Getenv("VAPID_SUBSCRIBER")
	if vapidSubscriber == "" {
		vapidSubscriber = "admin@example.com"
	}
	pushSender := push.NewSender(queries, vapidPub, vapidPriv, vapidSubscriber)
	pushHandler := push.NewHandler(queries, vapidPub)
	r.GET("/push/vapid-public-key", pushHandler.VAPIDPublicKey)
	r.POST("/push/subscribe", pushHandler.Subscribe)
	r.DELETE("/push/subscribe", pushHandler.Unsubscribe)

	r.GET("/settings", handler.Settings)

	h := handler.NewProjects(s)
	// Routes that do NOT require a project selection
	r.GET("/select-project", h.SelectProjectPage)
	r.POST("/select-project", h.SelectProject)
	r.POST("/switch-project", h.SwitchProject)
	r.GET("/projects", h.Index)
	r.GET("/projects/new", h.New)
	r.POST("/projects", h.Create)

	r.GET("/users", u.Index)
	r.GET("/users/new", u.New)
	r.POST("/users", u.Create)
	r.GET("/users/:id/edit", u.Edit)
	r.POST("/users/:id", u.Update)
	r.POST("/users/:id/delete", u.Delete)

	// Routes that require a project to be selected
	rp := r.Group("")
	rp.Use(authmw.RequireProject())

	rp.GET("/", func(c echo.Context) error {
		p := c.Get(authmw.CurrentProjectKey).(db.Project)
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", p.ID))
	})
	rp.GET("/projects/:id", h.Show)
	rp.GET("/projects/:id/edit", h.Edit)
	rp.POST("/projects/:id", h.Update)
	rp.POST("/projects/:id/delete", h.Delete)
	rp.GET("/projects/:id/test-github", h.TestGitHub)

	k := handler.NewKanban(s)
	rp.GET("/projects/:id/kanban/new", k.New)
	rp.POST("/projects/:id/kanban", k.Create)
	rp.POST("/projects/:id/kanban/:itemID/status", k.UpdateStatus)
	rp.POST("/projects/:id/kanban/:itemID/delete", k.Delete)
	rp.POST("/projects/:id/kanban/done/delete-all", k.DeleteAllDone)

	chat := handler.NewChatChannel(queries, pushSender)
	r.GET("/projects/:id/ws", chat.HandleWS)
	r.GET("/projects/:id/chat/history", chat.HandleHistory)

	commits := handler.NewCommits(s)
	r.GET("/projects/:id/commits", commits.List)

	return e, nil
}
