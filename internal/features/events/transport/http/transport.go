package http

import (
	"net/http"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http/create"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http/delete"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http/get"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http/list"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http/update"
)

type EventsHTTPHandler struct {
	svc service.EventsService
}

func NewEventsHTTPHandler(svc service.EventsService) *EventsHTTPHandler {
	return &EventsHTTPHandler{svc: svc}
}

func (h *EventsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodPost, "/events", h.Create),
		core_http_server.NewRoute(http.MethodGet, "/events", h.List),
		core_http_server.NewRoute(http.MethodGet, "/events/{id}", h.Get),
		core_http_server.NewRoute(http.MethodPatch, "/events/{id}", h.Update),
		core_http_server.NewRoute(http.MethodDelete, "/events/{id}", h.Delete),
	}
}

func (h *EventsHTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	create.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *EventsHTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	list.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *EventsHTTPHandler) Get(w http.ResponseWriter, r *http.Request) {
	get.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *EventsHTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	update.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *EventsHTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	delete.NewHandler(h.svc).ServeHTTP(w, r)
}
