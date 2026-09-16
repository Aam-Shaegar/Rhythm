package core_http_middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"
)

var errTest = errors.New("test unauthorized")
var errTestUnauthorized = errors.Join(errors.New("bad token"), core_errors.ErrUnauthorized)

func testLogger() *core_logger.Logger {
	cfg := core_logger.LoggerConfig{Folder: "/tmp", Level: "ERROR"}
	l, _ := core_logger.NewLogger(cfg)
	return l
}

func TestAuth_MissingHeader(t *testing.T) {
	h := Auth(func(s string) (string, string, error) { return "", "", nil })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
	)
	log := testLogger()
	defer log.Close()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(core_logger.ToContext(r.Context(), log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_BadFormat(t *testing.T) {
	for _, hdr := range []string{"Bearer", "Token abc", "Basic xyz"} {
		h := Auth(func(s string) (string, string, error) { return "u", "n", nil })(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
		)
		log := testLogger()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", hdr)
		r = r.WithContext(core_logger.ToContext(r.Context(), log))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		// plain "Bearer" -> 401 (no token part); other schemes -> 401
		if w.Code != 401 {
			t.Fatalf("header %q: expected 401, got %d", hdr, w.Code)
		}
		log.Close()
	}
	// "Bearer " with blank token passes format check but must be rejected by validator (real JWT does).
	// Mock a strict validator that rejects blank tokens.
	h := Auth(func(s string) (string, string, error) {
		if s == "" || s == " " {
			return "", "", errTestUnauthorized
		}
		return "u", "n", nil
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	log := testLogger()
	defer log.Close()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer  ")
	r = r.WithContext(core_logger.ToContext(r.Context(), log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("blank token: expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	h := Auth(func(s string) (string, string, error) {
		return "", "", errTestUnauthorized
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	log := testLogger()
	defer log.Close()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer bad.token.here")
	r = r.WithContext(core_logger.ToContext(r.Context(), log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d body %s", w.Code, w.Body.String())
	}
}

func TestAuth_Valid_SetsBothKeys(t *testing.T) {
	var gotTyped, gotString, gotUsername string
	var okTyped, okString bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTyped, okTyped = GetUserID(r.Context())
		// legacy string key must also be present (fixed for old handlers)
		v := r.Context().Value("user_id")
		if s, ok := v.(string); ok {
			gotString, okString = s, true
		}
		u, _ := GetUsername(r.Context())
		gotUsername = u
		w.WriteHeader(200)
	})
	h := Auth(func(s string) (string, string, error) {
		if s != "good" {
			t.Fatalf("unexpected token %s", s)
		}
		return "uid-123", "alice", nil
	})(next)
	log := testLogger()
	defer log.Close()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer good")
	r = r.WithContext(core_logger.ToContext(r.Context(), log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d %s", w.Code, w.Body.String())
	}
	if !okTyped || gotTyped != "uid-123" {
		t.Fatalf("typed key missing: %v %q", okTyped, gotTyped)
	}
	if !okString || gotString != "uid-123" {
		t.Fatalf("legacy string key missing: %v %q", okString, gotString)
	}
	if gotUsername != "alice" {
		t.Fatalf("username missing: %q", gotUsername)
	}
}

func TestAuth_CaseInsensitiveBearer(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := Auth(func(s string) (string, string, error) { return "u", "n", nil })(next)
	log := testLogger()
	defer log.Close()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "bearer good")
	r = r.WithContext(core_logger.ToContext(r.Context(), log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("bearer should be case-insensitive, got %d", w.Code)
	}
}
