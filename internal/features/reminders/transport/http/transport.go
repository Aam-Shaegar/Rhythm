package http

import (
	"net/http"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/transport/http/push"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/transport/http/settings"
)

type RemindersHTTPHandler struct {
	svc service.RemindersService
}

func NewRemindersHTTPHandler(svc service.RemindersService) *RemindersHTTPHandler {
	return &RemindersHTTPHandler{svc: svc}
}

func (h *RemindersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodGet, "/reminders/settings", h.Settings),
		core_http_server.NewRoute(http.MethodPatch, "/reminders/settings", h.UpdateSettings),
		core_http_server.NewRoute(http.MethodPost, "/reminders/push/subscriptions", h.SubscribePush),
		core_http_server.NewRoute(http.MethodDelete, "/reminders/push/subscriptions", h.UnsubscribePush),
		core_http_server.NewRoute(http.MethodGet, "/reminders/push/vapid-key", h.VapidKey),
	}
}

func (h *RemindersHTTPHandler) Settings(w http.ResponseWriter, r *http.Request) {
	settings.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *RemindersHTTPHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	settings.NewUpdateHandler(h.svc).ServeHTTP(w, r)
}

func (h *RemindersHTTPHandler) SubscribePush(w http.ResponseWriter, r *http.Request) {
	push.NewSubscribeHandler(h.svc).ServeHTTP(w, r)
}

func (h *RemindersHTTPHandler) UnsubscribePush(w http.ResponseWriter, r *http.Request) {
	push.NewUnsubscribeHandler(h.svc).ServeHTTP(w, r)
}

func (h *RemindersHTTPHandler) VapidKey(w http.ResponseWriter, r *http.Request) {
	push.NewVapidKeyHandler(h.svc).ServeHTTP(w, r)
}
