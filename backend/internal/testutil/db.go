// Package testutil provides the guarded integration-test database and
// synthetic fixtures. It only ever touches the throwaway compose MariaDB.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Facility codes seeded by ResetData.
const (
	FacilitySaku     = "saku-no-ie"    // max 4, Asia/Tokyo
	FacilityHonolulu = "test-honolulu" // max 2, Pacific/Honolulu
)

var (
	once     sync.Once
	sharedDB *gorm.DB
	lockConn *sql.Conn
	openErr  error
)

// AssertSafeTestDSN panics unless dsn points at the compose MariaDB
// (127.0.0.1:${TEST_DB_PORT:-33306}) and DB lodging_test or lodging_e2e.
// It never prints the DSN.
func AssertSafeTestDSN(dsn string) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		panic("testutil: TEST_DATABASE_URL is not a valid go-sql-driver DSN")
	}
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = "33306"
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:"+port {
		panic("testutil: refusing TEST_DATABASE_URL: host must be 127.0.0.1:" + port + " (compose.test.yaml)")
	}
	if cfg.DBName != "lodging_test" && cfg.DBName != "lodging_e2e" {
		panic("testutil: refusing TEST_DATABASE_URL: database must be lodging_test or lodging_e2e")
	}
}

// RequireDB returns the shared test DB with freshly reset data, or skips
// the test loudly when TEST_DATABASE_URL is unset.
//
// The first call takes the advisory lock GET_LOCK('sakunoie_test_lock') on a
// dedicated connection that lives until the test binary exits, so packages
// serialize even if someone forgets `go test -p 1`. DB tests must not call
// t.Parallel().
func RequireDB(t testing.TB) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SKIP: integration test needs TEST_DATABASE_URL (compose lodging_test; see backend/.env.test.example)")
	}
	AssertSafeTestDSN(dsn)
	once.Do(func() {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		sharedDB, openErr = database.Open(dsn, logger)
		if openErr != nil {
			return
		}
		sqlDB, err := sharedDB.DB()
		if err != nil {
			openErr = err
			return
		}
		ctx := context.Background()
		lockConn, openErr = sqlDB.Conn(ctx)
		if openErr != nil {
			return
		}
		var got sql.NullInt64
		if openErr = lockConn.QueryRowContext(ctx, "SELECT GET_LOCK('sakunoie_test_lock', 120)").Scan(&got); openErr != nil {
			return
		}
		if !got.Valid || got.Int64 != 1 {
			openErr = fmt.Errorf("testutil: could not take sakunoie_test_lock")
		}
	})
	if openErr != nil {
		t.Fatalf("testutil: open test DB: %v", openErr)
	}
	ResetData(t, sharedDB)
	return sharedDB
}

// ResetData deletes all rows in FK order and seeds two facilities.
func ResetData(t testing.TB, db *gorm.DB) {
	t.Helper()
	for _, table := range []string{
		"audit_logs", "registration_access_tokens", "guests", "import_rows",
		"guest_registrations", "admin_facility_access", "admin_users",
		"survey_responses", "api_idempotency_keys", "import_batches", "facilities",
	} {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("reset %s: %v", table, err)
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, f := range []entities.Facility{
		{Code: FacilitySaku, Name: "佐久の家 (test)", Timezone: "Asia/Tokyo", MaxGuests: 4, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{Code: FacilityHonolulu, Name: "Test Honolulu", Timezone: "Pacific/Honolulu", MaxGuests: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&f).Error; err != nil {
			t.Fatalf("seed facility: %v", err)
		}
	}
}

// CountRows returns COUNT(*) of a table.
func CountRows(t testing.TB, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
