package core_http_request

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	users_dtos "github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
)

func TestDecodeRegister_Valid(t *testing.T) {
	body := `{"username":"alice","email":"alice@example.com","password":"password123"}`
	r := httptest.NewRequest("POST", "/auth/register", strings.NewReader(body))
	var in users_dtos.RegisterInput
	if err := DecodeAndValidateRequest(r, &in); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestDecodeRegister_EdgeCases(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"short username", `{"username":"ab","email":"a@b.com","password":"password123"}`},
		{"bad email", `{"username":"alice","email":"notanemail","password":"password123"}`},
		{"short password", `{"username":"alice","email":"a@b.com","password":"123"}`},
		{"too long password", `{"username":"alice","email":"a@b.com","password":"` + strings.Repeat("x", 73) + `"}`},
		{"missing password", `{"username":"alice","email":"a@b.com"}`},
		{"invalid json", `{not json`},
		{"empty json", ``},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(c.body))
			var in users_dtos.RegisterInput
			if err := DecodeAndValidateRequest(r, &in); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestDecodeLogin_EdgeCases(t *testing.T) {
	valid := `{"email":"a@b.com","password":"secret123"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(valid))
	var in users_dtos.LoginInput
	if err := DecodeAndValidateRequest(r, &in); err != nil {
		t.Fatalf("valid login rejected: %v", err)
	}
	invalid := []string{
		`{"email":"bad","password":"x"}`,
		`{"email":"a@b.com"}`,
		`{"password":"x"}`,
		`{}`,
	}
	for _, b := range invalid {
		r := httptest.NewRequest("POST", "/", strings.NewReader(b))
		var in users_dtos.LoginInput
		if err := DecodeAndValidateRequest(r, &in); err == nil {
			t.Fatalf("expected error for %s", b)
		}
	}
}

func TestDecodeTask_EdgeCases(t *testing.T) {
	valid := `{"title":"t","due_at":"2026-09-20T10:00:00Z"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(valid))
	var in tasks_domain.CreateTaskInput
	if err := DecodeAndValidateRequest(r, &in); err != nil {
		t.Fatalf("valid task rejected: %v", err)
	}
	cases := []string{
		`{"title":"","due_at":"2026-09-20T10:00:00Z"}`,
		`{"title":"t","due_at":"not-a-date"}`,
		`{"title":"t","due_at":"2026-09-20"}`,
		`{"due_at":"2026-09-20T10:00:00Z"}`,
		`{"title":"` + strings.Repeat("x", 201) + `","due_at":"2026-09-20T10:00:00Z"}`,
		`{"title":"t","due_at":"2026-09-20T10:00:00Z","recurrence_type":"hourly"}`,
		`{"title":"t","due_at":"2026-09-20T10:00:00Z","recurrence_end":"bad"}`,
		`{"title":"t","due_at":"2026-09-20T10:00:00Z","description":"` + strings.Repeat("x", 2001) + `"}`,
	}
	for _, b := range cases {
		r := httptest.NewRequest("POST", "/", strings.NewReader(b))
		var in tasks_domain.CreateTaskInput
		if err := DecodeAndValidateRequest(r, &in); err == nil {
			t.Fatalf("expected error for %s", b[:60])
		}
	}
	// recurrence valid values
	for _, rt := range []string{"daily", "weekly", "monthly", "yearly"} {
		b := `{"title":"t","due_at":"2026-09-20T10:00:00Z","recurrence_type":"` + rt + `","recurrence_end":"2026-10-20T10:00:00Z"}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(b))
		var in tasks_domain.CreateTaskInput
		if err := DecodeAndValidateRequest(r, &in); err != nil {
			t.Fatalf("recurrence %s rejected: %v", rt, err)
		}
	}
}

func TestDecodeQuery_TimeBoolInt(t *testing.T) {
	// RFC3339
	r := httptest.NewRequest("GET", "/?date=2026-09-20T10:00:00Z", nil)
	var q domain.DailyReportQuery
	if err := DecodeQueryParams(r, &q); err != nil {
		t.Fatalf("RFC3339 date rejected: %v", err)
	}
	if q.Date.Day() != 20 {
		t.Fatalf("wrong date parsed: %v", q.Date)
	}
	// YYYY-MM-DD
	r = httptest.NewRequest("GET", "/?date=2026-09-20", nil)
	q = domain.DailyReportQuery{}
	if err := DecodeQueryParams(r, &q); err != nil {
		t.Fatalf("YYYY-MM-DD rejected: %v", err)
	}
	// invalid date -> 400 (must wrap ErrInvalidArgument)
	r = httptest.NewRequest("GET", "/?date=not-a-date", nil)
	q = domain.DailyReportQuery{}
	if err := DecodeQueryParams(r, &q); err == nil {
		t.Fatalf("expected error for bad date")
	}
	// empty query -> zero, no error
	r = httptest.NewRequest("GET", "/", nil)
	q = domain.DailyReportQuery{}
	if err := DecodeQueryParams(r, &q); err != nil {
		t.Fatalf("empty query should be ok: %v", err)
	}
	if !q.Date.IsZero() {
		t.Fatalf("expected zero date")
	}
	// bool + int + time pointer in TaskFilter
	r = httptest.NewRequest("GET", "/?completed=true&recurring=false&limit=10&offset=5&date_from=2026-01-01&date_to=2026-12-31", nil)
	var f tasks_domain.TaskFilter
	if err := DecodeQueryParams(r, &f); err != nil {
		t.Fatalf("task filter rejected: %v", err)
	}
	if f.Completed == nil || *f.Completed != true {
		t.Fatalf("completed not parsed")
	}
	if f.Recurring == nil || *f.Recurring != false {
		t.Fatalf("recurring not parsed")
	}
	if f.Limit != 10 || f.Offset != 5 {
		t.Fatalf("limit/offset not parsed: %+v", f)
	}
	if f.DateFrom == nil || f.DateTo == nil {
		t.Fatalf("dates not parsed")
	}
	// invalid bool
	r = httptest.NewRequest("GET", "/?completed=maybe", nil)
	f = tasks_domain.TaskFilter{}
	if err := DecodeQueryParams(r, &f); err == nil {
		t.Fatalf("expected error for bad bool")
	}
	// invalid int
	r = httptest.NewRequest("GET", "/?limit=abc", nil)
	f = tasks_domain.TaskFilter{}
	if err := DecodeQueryParams(r, &f); err == nil {
		t.Fatalf("expected error for bad int")
	}
	// invalid time in filter
	r = httptest.NewRequest("GET", "/?date_from=baddate", nil)
	f = tasks_domain.TaskFilter{}
	if err := DecodeQueryParams(r, &f); err == nil {
		t.Fatalf("expected error for bad date_from")
	}
	// period query both dates
	r = httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-01-31", nil)
	var pq domain.PeriodReportQuery
	if err := DecodeQueryParams(r, &pq); err != nil {
		t.Fatalf("period query rejected: %v", err)
	}
	if pq.From.IsZero() || pq.To.IsZero() {
		t.Fatalf("period dates zero")
	}
	// duration still works (time.Duration field)
	type durQ struct {
		D time.Duration `query:"d"`
	}
	r = httptest.NewRequest("GET", "/?d=5s", nil)
	var dq durQ
	if err := DecodeQueryParams(r, &dq); err != nil {
		t.Fatalf("duration rejected: %v", err)
	}
	if dq.D != 5*time.Second {
		t.Fatalf("wrong duration %v", dq.D)
	}
}

func TestDecodeQuery_NonPointer(t *testing.T) {
	var f tasks_domain.TaskFilter
	r := httptest.NewRequest("GET", "/", nil)
	if err := DecodeQueryParams(r, f); err == nil {
		t.Fatalf("expected error for non-pointer dest")
	}
}
