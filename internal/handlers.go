package internal

import (
	"encoding/json"
	"net/http"
	"path"

	"github.com/sirupsen/logrus"
)

type API struct {
	Manager *TaskManager
}

func (api *API) CreateTask(w http.ResponseWriter, r *http.Request) {
	id, err := api.Manager.Create()
	if err != nil {
		Logger.WithError(err).Error("failed to create task")
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"task_id": id})
}

func (api *API) AddLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
		URL    string `json:"url"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := api.Manager.AddURL(req.TaskID, req.URL); err != nil {
		Logger.WithError(err).WithField("task_id", req.TaskID).Error("failed to add link")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Logger.WithFields(logrus.Fields{"task_id": req.TaskID, "url": req.URL}).Info("link added")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (api *API) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := api.Manager.List()
	resp := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, map[string]interface{}{
			"id":     t.ID,
			"status": t.Status,
			"errors": t.Errors,
		})
	}
	Logger.Info("tasks listed")
	json.NewEncoder(w).Encode(resp)
}

func (api *API) ForceZip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := api.Manager.ForceZip(req.TaskID); err != nil {
		Logger.WithError(err).WithField("task_id", req.TaskID).Error("force zip failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Logger.WithField("task_id", req.TaskID).Info("force zip started")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

func (api *API) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	if err := api.Manager.Delete(id); err != nil {
		Logger.WithError(err).WithField("task_id", id).Error("delete task failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Logger.WithField("task_id", id).Info("task deleted via API")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (api *API) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	task, err := api.Manager.Status(id)
	if err != nil {
		Logger.WithError(err).WithField("task_id", id).Error("status request failed")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	Logger.WithField("task_id", id).Info("status requested")
	out := map[string]interface{}{"status": task.Status, "errors": task.Errors}
	if task.Status == StatusComplete {
		out["archive_url"] = "/download/" + id
	}
	json.NewEncoder(w).Encode(out)
}

func (api *API) Download(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	task, err := api.Manager.Status(id)
	if err != nil || task.Status != StatusComplete {
		Logger.WithField("task_id", id).Error("download requested before ready")
		http.Error(w, "not ready", http.StatusBadRequest)
		return
	}
	Logger.WithField("task_id", id).Info("download started")
	http.ServeFile(w, r, task.ZipPath)
}
