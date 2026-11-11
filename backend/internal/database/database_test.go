package database

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPrepareDSNForcesUTC(t *testing.T) {
	cfg, overridden, err := PrepareDSN("user:secret-pw@tcp(127.0.0.1:1)/db?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		t.Fatal(err)
	}
	if !overridden {
		t.Fatal("loc=Local must be reported as overridden")
	}
	if cfg.Loc != time.UTC || !cfg.ParseTime || cfg.Params["time_zone"] != "'+00:00'" {
		t.Fatalf("loc=%v parseTime=%v time_zone=%q", cfg.Loc, cfg.ParseTime, cfg.Params["time_zone"])
	}
	if !strings.Contains(cfg.FormatDSN(), "charset=utf8mb4") {
		t.Fatal("other params must be kept")
	}
}

func TestPrepareDSNAlreadyUTC(t *testing.T) {
	_, overridden, err := PrepareDSN("u:p@tcp(127.0.0.1:1)/db?parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27")
	if err != nil {
		t.Fatal(err)
	}
	if overridden {
		t.Fatal("UTC DSN must not be reported as overridden")
	}
}

func TestPrepareDSNInvalidDoesNotLeak(t *testing.T) {
	_, _, err := PrepareDSN("user:secret-pw@bogus(")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-pw") {
		t.Fatal("error leaks DSN")
	}
}

func TestOpenWarnsOnceWithoutDSN(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	// Opening does not connect (no ping), so an unreachable port is fine.
	db, err := Open("user:secret-pw@tcp(127.0.0.1:1)/db?loc=Local", logger)
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	out := buf.String()
	if strings.Count(out, "db: loc overridden to UTC per contract §2") != 1 {
		t.Fatalf("expected exactly one WARN, got %q", out)
	}
	if !strings.Contains(out, `"level":"WARN"`) {
		t.Fatalf("expected WARN level, got %q", out)
	}
	if strings.Contains(out, "secret-pw") || strings.Contains(out, "127.0.0.1") {
		t.Fatal("log leaks DSN")
	}
}
