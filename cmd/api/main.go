package main

import (
	"fmt"
	"linkzipper/internal"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := internal.Load()
	internal.InitLogger(cfg.Logging.Level, cfg.Logging.File)
	mgr := internal.NewManager(
		cfg.Limits.MaxTasks,
		cfg.Limits.MaxFilesPerTask,
		cfg.Limits.AllowedExts,
	)
	api := &internal.API{Manager: mgr}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			internal.Logger.WithFields(logrus.Fields{
				"remote": r.RemoteAddr,
				"method": r.Method,
				"path":   r.URL.Path,
			}).Info("request")
			next.ServeHTTP(w, r)
		})
	})
	r.Post("/tasks", api.CreateTask)
	r.Post("/tasks/links", api.AddLink)
	r.Get("/tasks/status/*", api.GetStatus)
	r.Get("/download/*", api.Download)

	r.Mount("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("testdata")),
		),
	)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	internal.Logger.Infof("Starting server on %s", addr)
	internal.Logger.Fatal(http.ListenAndServe(addr, r))
}
