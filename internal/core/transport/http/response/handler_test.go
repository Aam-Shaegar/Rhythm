package core_http_response

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	core_error "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
)

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
		text string
	}{
		{core_error.ErrInvalidArgument, 400, "Bad Request"},
		{core_error.ErrNotFound, 404, "Not Found"},
		{core_error.ErrConflict, 409, "Conflict"},
		{core_error.ErrUnauthorized, 401, "Unauthorized"},
		{errors.New("boom"), 500, "Internal Server Error"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		// nil logger must NOT panic (regression test for 500-on-error bug)
		NewHTTPResponseHandler(nil, w).ErrorResponse(c.err, "msg")
		if w.Code != c.code {
			t.Fatalf("err %v: expected %d got %d", c.err, c.code, w.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("body not json: %v", err)
		}
		if body["message"] != "msg" {
			t.Fatalf("message lost: %v", body)
		}
		if body["error"] != c.text {
			t.Fatalf("expected error text %q got %q", c.text, body["error"])
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("content-type %q", ct)
		}
	}
}

func TestWrappedErrors_MapCorrectly(t *testing.T) {
	w := httptest.NewRecorder()
	wrapped := errors.Join(errors.New("ctx"), core_error.ErrNotFound)
	NewHTTPResponseHandler(nil, w).ErrorResponse(wrapped, "m")
	if w.Code != 404 {
		t.Fatalf("wrapped ErrNotFound should be 404, got %d", w.Code)
	}
}

func TestJSONResponse_Shape(t *testing.T) {
	w := httptest.NewRecorder()
	NewHTTPResponseHandler(nil, w).JSONResponse(map[string]int{"a": 1}, 201)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NewHTTPResponseHandler(nil, w).NoContentResponse()
	if w.Code != 204 {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

func TestPanicResponse_NilLogger(t *testing.T) {
	w := httptest.NewRecorder()
	NewHTTPResponseHandler(nil, w).PanicResponse("oops", "panic msg")
	if w.Code != 500 {
		t.Fatalf("expected 500 got %d", w.Code)
	}
}
