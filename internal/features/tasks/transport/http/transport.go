package http

import (
	"net/http"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/create"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/get"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/list"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/update"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/delete"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http/complete"
)

type TasksHTTPHandler struct {
	svc service.TasksService
}

func NewTasksHTTPHandler(svc service.TasksService) *TasksHTTPHandler {
	return &TasksHTTPHandler{svc: svc}
}

func (h *TasksHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodPost, "/tasks", h.Create),
		core_http_server.NewRoute(http.MethodGet, "/tasks", h.List),
		core_http_server.NewRoute(http.MethodGet, "/tasks/{id}", h.Get),
		core_http_server.NewRoute(http.MethodPatch, "/tasks/{id}", h.Update),
		core_http_server.NewRoute(http.MethodPatch, "/tasks/{id}/complete", h.Complete),
		core_http_server.NewRoute(http.MethodDelete, "/tasks/{id}", h.Delete),
	}
}

func (h *TasksHTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	create.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *TasksHTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	list.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *TasksHTTPHandler) Get(w http.ResponseWriter, r *http.Request) {
	get.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *TasksHTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	update.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *TasksHTTPHandler) Complete(w http.ResponseWriter, r *http.Request) {
	complete.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *TasksHTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	delete.NewHandler(h.svc).ServeHTTP(w, r)
}