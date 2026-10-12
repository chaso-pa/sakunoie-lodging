// Package testutil provides the MySQL database used by integration tests.
package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/database"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// defaultDSN points at the MySQL container from the repository's
// docker-compose.yml. Override it with TEST_MYSQL_DSN; the database name in
// the DSN is ignored.
const defaultDSN = "root:root@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local"

// DB is a freshly migrated test database.
type DB struct {
	Gorm *gorm.DB
	// Sessions is the pool for the scs session store (see
	// database.OpenSessionDB).
	Sessions *sql.DB
}

// NewDB drops and recreates the database sakunoie_test_<name>, applies the
// Prisma migrations from prisma/migrations to it and connects.
//
// Each test package must pass its own name: packages run in parallel and
// would otherwise drop each other's tables.
//
// When MySQL is unreachable the test is skipped, or failed if
// REQUIRE_TEST_DB=1 is set.
func NewDB(t testing.TB, name string) *DB {
	t.Helper()

	base := os.Getenv("TEST_MYSQL_DSN")
	if base == "" {
		base = defaultDSN
	}
	cfg, err := mysqldriver.ParseDSN(base)
	if err != nil {
		t.Fatalf("parse TEST_MYSQL_DSN: %v", err)
	}
	dbName := "sakunoie_test_" + name

	adminCfg := cfg.Clone()
	adminCfg.DBName = ""
	adminCfg.MultiStatements = true
	admin, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer admin.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		msg := "MySQL is not reachable (" + err.Error() + "). Start it with `docker compose up -d mysql` at the repository root, or set TEST_MYSQL_DSN."
		if os.Getenv("REQUIRE_TEST_DB") == "1" {
			t.Fatal(msg)
		}
		t.Skip("skipping integration test: " + msg)
	}

	if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + dbName + "`; CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; USE `" + dbName + "`"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	// Run the migrations on one connection so USE stays in effect.
	conn, err := admin.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "USE `"+dbName+"`"); err != nil {
		t.Fatalf("select test database: %v", err)
	}
	for _, file := range migrationFiles(t) {
		sqlText, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration: %v", err)
		}
		if _, err := conn.ExecContext(context.Background(), string(sqlText)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}

	testCfg := cfg.Clone()
	testCfg.DBName = dbName
	dsn := testCfg.FormatDSN()

	gormDB, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sessionDB, err := database.OpenSessionDB(dsn)
	if err != nil {
		t.Fatalf("open session database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			sqlDB.Close()
		}
		sessionDB.Close()
	})

	return &DB{Gorm: gormDB, Sessions: sessionDB}
}

// migrationFiles returns prisma/migrations/*/migration.sql in order.
func migrationFiles(t testing.TB) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testutil source file")
	}
	backendRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	files, err := filepath.Glob(filepath.Join(backendRoot, "prisma", "migrations", "*", "migration.sql"))
	if err != nil {
		t.Fatalf("find migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no Prisma migrations found under prisma/migrations")
	}
	sort.Strings(files)
	return files
}
