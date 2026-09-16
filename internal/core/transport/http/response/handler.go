package core_http_response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	core_error "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"

	"go.uber.org/zap"
)

type HTTPResponseHandler struct {
	log *core_logger.Logger
	rw  http.ResponseWriter
}

func NewHTTPResponseHandler(log *core_logger.Logger, rw http.ResponseWriter) *HTTPResponseHandler {
	return &HTTPResponseHandler{
		log: log,
		rw:  rw,
	}
}

func (h *HTTPResponseHandler) NoContentResponse() {
	h.rw.WriteHeader(http.StatusNoContent)
}

func (h *HTTPResponseHandler) ErrorResponse(err error, msg string) {
	var (
		statusCode int
		logFunc    func(string, ...zap.Field)
	)

	switch {
	case errors.Is(err, core_error.ErrInvalidArgument):
		statusCode = http.StatusBadRequest
		if h.log != nil {
			logFunc = h.log.Warn
		}
	case errors.Is(err, core_error.ErrNotFound):
		statusCode = http.StatusNotFound
		if h.log != nil {
			logFunc = h.log.Debug
		}
	case errors.Is(err, core_error.ErrConflict):
		statusCode = http.StatusConflict
		if h.log != nil {
			logFunc = h.log.Warn
		}
	case errors.Is(err, core_error.ErrUnauthorized):
		statusCode = http.StatusUnauthorized
		if h.log != nil {
			logFunc = h.log.Warn
		}
	default:
		statusCode = http.StatusInternalServerError
		if h.log != nil {
			logFunc = h.log.Error
		}
	}
	if logFunc != nil {
		logFunc(msg, zap.Error(err))
	}
	h.errorResponse(statusCode, err, msg)
}

func (h *HTTPResponseHandler) PanicResponse(p any, msg string) {
	statusCode := http.StatusInternalServerError
	err := fmt.Errorf("unexpected panic: %v", p)

	if h.log != nil {
		h.log.Error(msg, zap.Error(err))
	}
	h.errorResponse(statusCode, err, msg)
}

func (h *HTTPResponseHandler) errorResponse(statusCode int, err error, msg string) {
	response := map[string]string{
		"message": msg,
		"error":   http.StatusText(statusCode),
	}
	h.JSONResponse(response, statusCode)
}

func (h *HTTPResponseHandler) JSONResponse(responseBody any, statusCode int) {
	h.rw.Header().Set("Content-Type", "application/json")
	h.rw.WriteHeader(statusCode)
	if err := json.NewEncoder(h.rw).Encode(responseBody); err != nil {
		if h.log != nil {
			h.log.Error("write HTTP response", zap.Error(err))
		}
	}
}
