package postgres

import (
	"errors"
	"testing"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool"
)

type fakeRow struct{ err error }

func (f fakeRow) Scan(dest ...any) error { return f.err }

func TestScanRefreshToken_NoRows_MapsToNotFound(t *testing.T) {
	_, err := scanRefreshToken(fakeRow{err: core_pool.ErrNoRows})
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestScanRefreshToken_OtherError_Passthrough(t *testing.T) {
	boom := errors.New("boom")
	_, err := scanRefreshToken(fakeRow{err: boom})
	if !errors.Is(err, boom) {
		t.Fatalf("expected passthrough, got %v", err)
	}
}
