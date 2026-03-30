package main

import (
	"crypto/md5"
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
	"rustymanager/internal/filestore"
	"rustymanager/internal/handler"
	mcphandler "rustymanager/internal/mcp"
	authmw "rustymanager/internal/middleware"
	"rustymanager/internal/push"
	"rustymanager/internal/store"
	"rustymanager/web"
)

type renderer struct {
	fsys       fs.FS
	base       *template.Template
	cssVersion string
	swVersion  string
}

func newRenderer(fsys fs.FS) (*renderer, error) {
	base, err := template.ParseFS(fsys, "templates/layout.html")
	if err != nil {
		return nil, err
	}
	return &renderer{
		fsys:       fsys,
		base:       base,
		cssVersion: assetVersion(fsys, "static/css/output.css"),
		swVersion:  assetVersion(fsys, "static/sw.js"),
	}, nil
}

func assetVersion(fsys fs.FS, path string) string {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "1"
	}
	sum := md5.Sum(data)
	return fmt.Sprintf("%x", sum[:4])
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
		m["CSSVersion"] = r.cssVersion
		m["SWVersion"] = r.swVersion
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

	a := handler.NewAuth(s)
	e.GET("/login", a.LoginPage)
	e.POST("/login", a.Login)
	e.GET("/register", a.RegisterPage)
	e.POST("/register", a.Register)

	// Auth-protected group
	p := e.Group("")
	p.Use(authmw.RequireAuth(s))
	p.Use(authmw.LoadProject(s))
	p.POST("/logout", a.Logout)

	vapidPub, vapidPriv := loadVAPIDKeys()
	vapidSubscriber := os.Getenv("VAPID_SUBSCRIBER")
	if vapidSubscriber == "" {
		vapidSubscriber = "admin@example.com"
	}
	pushSender := push.NewSender(queries, vapidPub, vapidPriv, vapidSubscriber)
	pushHandler := push.NewHandler(queries, vapidPub)
	p.GET("/push/vapid-public-key", pushHandler.VAPIDPublicKey)
	p.POST("/push/subscribe", pushHandler.Subscribe)
	p.DELETE("/push/subscribe", pushHandler.Unsubscribe)

	settings := handler.NewSettings(s)
	p.GET("/settings", settings.Index)
	p.GET("/settings/admin", settings.Admin)
	p.POST("/settings/admin/users/:id/reset-password", settings.ResetPassword)
	p.POST("/settings/admin/users/:id/delete", settings.DeleteUser)
	p.POST("/settings/api-token", settings.GenerateAPIToken)
	p.POST("/settings/api-token/revoke", settings.RevokeAPIToken)

	h := handler.NewProjects(s)
	p.GET("/select-project", h.SelectProjectPage)
	p.POST("/select-project", h.SelectProject)
	p.POST("/switch-project", h.SwitchProject)
	p.GET("/projects", h.Index)
	p.GET("/projects/new", h.New)
	p.POST("/projects", h.Create)

	u := handler.NewUsers(s)
	p.GET("/users", u.Index)
	p.GET("/users/:id/edit", u.Edit)
	p.POST("/users/:id", u.Update)
	p.POST("/users/:id/delete", u.Delete)

	// Routes that require a project to be selected
	rp := p.Group("")
	rp.Use(authmw.RequireProject())

	rp.GET("/", func(c echo.Context) error {
		proj := c.Get(authmw.CurrentProjectKey).(db.Project)
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", proj.ID))
	})
	rp.GET("/projects/:id", h.Show)
	rp.GET("/projects/:id/edit", h.Edit)
	rp.POST("/projects/:id", h.Update)
	rp.POST("/projects/:id/delete", h.Delete)

	k := handler.NewKanban(s)
	rp.GET("/projects/:id/kanban/new", k.New)
	rp.POST("/projects/:id/kanban", k.Create)
	rp.POST("/projects/:id/kanban/:itemID/status", k.UpdateStatus)
	rp.POST("/projects/:id/kanban/:itemID/delete", k.Delete)
	rp.POST("/projects/:id/kanban/done/delete-all", k.DeleteAllDone)

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "uploads"
	}
	uploadStore, err := filestore.New(uploadsDir)
	if err != nil {
		return nil, fmt.Errorf("filestore: %w", err)
	}
	e.Static("/uploads", uploadsDir)
	p.POST("/projects/:id/chat/upload", filestore.NewHandler(uploadStore).Upload)

	chat := handler.NewChatChannel(queries, pushSender)
	p.GET("/projects/:id/ws", chat.HandleWS)
	p.GET("/projects/:id/chat/history", chat.HandleHistory)

	jaas, err := handler.NewJaaSHandler()
	if err != nil {
		return nil, fmt.Errorf("JaaS: %w", err)
	}
	p.GET("/call/token", jaas.Token)

	// MCP server — authenticated by Bearer API token
	e.Any("/mcp", echo.WrapHandler(mcphandler.Handler(s)))

	return e, nil
}
