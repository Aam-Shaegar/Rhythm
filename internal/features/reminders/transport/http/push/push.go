package push

import (
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_request "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/request"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	"github.com/google/uuid"
)

type SubscribeHandler struct {
	svc service.RemindersService
}

func NewSubscribeHandler(svc service.RemindersService) *SubscribeHandler {
	return &SubscribeHandler{svc: svc}
}

func (h *SubscribeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	var input domain.PushSubscriptionInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	if err := h.svc.SavePushSubscription(r.Context(), userID, input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to save subscription")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(map[string]string{"status": "subscribed"}, http.StatusOK)
}

type UnsubscribeHandler struct {
	svc service.RemindersService
}

func NewUnsubscribeHandler(svc service.RemindersService) *UnsubscribeHandler {
	return &UnsubscribeHandler{svc: svc}
}

func (h *UnsubscribeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	var input domain.DeletePushSubscriptionInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	if err := h.svc.DeletePushSubscription(r.Context(), userID, input.Endpoint); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to delete subscription")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(map[string]string{"status": "unsubscribed"}, http.StatusOK)
}

type VapidKeyHandler struct {
	svc service.RemindersService
}

func NewVapidKeyHandler(svc service.RemindersService) *VapidKeyHandler {
	return &VapidKeyHandler{svc: svc}
}

func (h *VapidKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := userIDFromContext(r); !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	key := h.svc.PushPublicKey()
	if key == "" {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrNotFound, "push not configured")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(map[string]string{"public_key": key}, http.StatusOK)
}

func userIDFromContext(r *http.Request) (uuid.UUID, bool) {
	val := r.Context().Value("user_id")
	if val == nil {
		return uuid.Nil, false
	}
	idStr, ok := val.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
