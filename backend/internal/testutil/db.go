// Package testutil provides the MySQL database used by integration tests.
package testutil

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"syscall"
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
	// DSN connects to this database. A test can open further connections
	// with database.Open(DSN).
	DSN string
}

// dbPrefix starts the name of every database this package creates.
const dbPrefix = "sakunoie_test_"

// NewDB returns an empty, migrated database named
// sakunoie_test_<name>_<pid>.
//
// The first call in a process creates the database and applies the Prisma
// migrations from prisma/migrations; later calls empty its tables instead,
// which is much cheaper than recreating it for every test. The database is
// dropped when the package's tests finish, provided the package's TestMain
// calls Main.
//
// The process id is part of the name so that two runs of the suite at the
// same time (two shells, CI next to a developer) each get their own
// databases instead of dropping each other's. name keeps the packages of
// one run apart: they are separate processes already, but the name shows
// which package a database belongs to.
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
	dbName := dbPrefix + name + "_" + strconv.Itoa(os.Getpid())

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

	sweepOnce.Do(func() { sweepStaleDatabases(t, admin, cfg.Addr) })

	// One connection for the setup, so USE stays in effect.
	conn, err := admin.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	defer conn.Close()

	created.mu.Lock()
	exists := created.names[dbName] != ""
	created.mu.Unlock()
	if exists {
		emptyTables(t, conn, dbName)
	} else {
		createDatabase(t, conn, dbName)
		created.mu.Lock()
		created.names[dbName] = adminCfg.FormatDSN()
		created.mu.Unlock()
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

	return &DB{Gorm: gormDB, Sessions: sessionDB, DSN: dsn}
}

// created records the databases this process made, with the DSN to drop
// them through.
var created = struct {
	mu    sync.Mutex
	names map[string]string
}{names: map[string]string{}}

func createDatabase(t testing.TB, conn *sql.Conn, dbName string) {
	t.Helper()
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+dbName+"`; CREATE DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; USE `"+dbName+"`"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	for _, file := range migrationFiles(t) {
		sqlText, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration: %v", err)
		}
		if _, err := conn.ExecContext(ctx, string(sqlText)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}
}

// emptyTables deletes every row of every table, leaving the schema as the
// migrations made it.
func emptyTables(t testing.TB, conn *sql.Conn, dbName string) {
	t.Helper()
	ctx := context.Background()
	rows, err := conn.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE'", dbName)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("list tables: %v", err)
		}
		tables = append(tables, table)
	}
	rows.Close()

	// Foreign key checks are per connection, hence the single conn.
	statements := "USE `" + dbName + "`; SET FOREIGN_KEY_CHECKS = 0; "
	for _, table := range tables {
		statements += "DELETE FROM `" + table + "`; "
	}
	statements += "SET FOREIGN_KEY_CHECKS = 1"
	if _, err := conn.ExecContext(ctx, statements); err != nil {
		t.Fatalf("empty test database: %v", err)
	}
}

// Main runs a package's tests and then drops the databases NewDB created
// for it. Call it from TestMain in every package that uses NewDB:
//
//	func TestMain(m *testing.M) { testutil.Main(m) }
//
// Without it the databases stay until the next run on this machine sweeps
// them up.
func Main(m *testing.M) {
	code := m.Run()
	created.mu.Lock()
	for name, adminDSN := range created.names {
		if admin, err := sql.Open("mysql", adminDSN); err == nil {
			_, _ = admin.Exec("DROP DATABASE IF EXISTS `" + name + "`")
			admin.Close()
		}
	}
	created.mu.Unlock()
	os.Exit(code)
}

var (
	sweepOnce sync.Once
	// A name this package may have created: the old fixed form
	// sakunoie_test_<name>, or the current one ending in _<pid>.
	testDBName = regexp.MustCompile(`^sakunoie_test_[a-z_]+?(?:_([0-9]+))?$`)
)

// sweepStaleDatabases drops test databases left behind by runs that did not
// clean up: the fixed names used before the pid was added, and names whose
// process no longer exists (a run that was killed).
//
// It only acts when MySQL is on this machine. A pid says nothing about a
// process on another host, so against a shared server a live run's database
// could be mistaken for a dead one. Databases of a process that is still
// alive are never touched.
func sweepStaleDatabases(t testing.TB, admin *sql.DB, addr string) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return
	}

	rows, err := admin.Query(`SHOW DATABASES LIKE 'sakunoie\\_test\\_%'`)
	if err != nil {
		return
	}
	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			names = append(names, name)
		}
	}
	rows.Close()

	for _, name := range names {
		m := testDBName.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		if m[1] != "" {
			pid, err := strconv.Atoi(m[1])
			if err != nil || processAlive(pid) {
				continue
			}
		}
		if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + name + "`"); err == nil {
			t.Logf("dropped stale test database %s", name)
		}
	}
}

// processAlive reports whether a process with this id exists. Signal 0
// checks for existence without sending anything; EPERM means it exists but
// belongs to someone else.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
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
