package main

import (
	"fmt"
	"linkzipper/internal"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
    cfg := internal.Load()
    mgr := internal.NewManager(
        cfg.Limits.MaxTasks,
        cfg.Limits.MaxFilesPerTask,
        cfg.Limits.AllowedExts,
    )
    api := &internal.API{Manager: mgr}

    r := chi.NewRouter()
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
    log.Printf("Starting server on %s", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}
