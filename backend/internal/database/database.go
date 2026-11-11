// Package database opens the GORM connection with the UTC rules of the
// contract (§2): DATETIME(6) columns hold UTC, whatever the DSN says.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const utcTimeZone = "'+00:00'"

// PrepareDSN parses a go-sql-driver DSN and forces Loc=UTC, ParseTime=true
// and session time_zone='+00:00'. overridden reports whether the DSN asked
// for something else (e.g. loc=Local in the user's .env).
func PrepareDSN(dsn string) (cfg *mysql.Config, overridden bool, err error) {
	cfg, err = mysql.ParseDSN(dsn)
	if err != nil {
		// The driver's error may contain the DSN; do not wrap it.
		return nil, false, fmt.Errorf("database: invalid GORM_DATABASE_URL")
	}
	if cfg.Loc != time.UTC || cfg.Params["time_zone"] != utcTimeZone {
		overridden = true
	}
	cfg.Loc = time.UTC
	cfg.ParseTime = true
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["time_zone"] = utcTimeZone
	return cfg, overridden, nil
}

// Now is the GORM clock: UTC, microsecond precision (DATETIME(6)).
func Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// Open connects with the forced UTC settings. It logs one WARN when the DSN
// location was overridden; the DSN itself is never logged.
func Open(dsn string, logger *slog.Logger) (*gorm.DB, error) {
	cfg, overridden, err := PrepareDSN(dsn)
	if err != nil {
		return nil, err
	}
	if overridden {
		logger.Warn("db: loc overridden to UTC per contract §2")
	}
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("database: connector: %w", err)
	}
	sqlDB := sql.OpenDB(connector)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
		NowFunc:                Now,
		DisableAutomaticPing:   true, // callers Ping explicitly (main, /ready)
		SkipDefaultTransaction: true,
		Logger: gormlogger.New(slog.NewLogLogger(logger.Handler(), slog.LevelWarn), gormlogger.Config{
			SlowThreshold:             500 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true, // never log bound values (PII)
		}),
	})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: open: %w", err)
	}
	return db, nil
}

// Ping checks connectivity with a timeout (used by /ready).
func Ping(ctx context.Context, db *gorm.DB, timeout time.Duration) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return sqlDB.PingContext(ctx)
}
