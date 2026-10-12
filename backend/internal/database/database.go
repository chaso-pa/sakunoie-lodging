package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects GORM to MySQL. The schema is owned by Prisma migrations;
// AutoMigrate must never be called on this connection.
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// ParameterizedQueries keeps bound values out of the SQL that slow
		// query and error lines print: those values are post text, review
		// text and coordinates, none of which may reach the logs.
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// OpenSessionDB opens the connection pool used by the scs session store.
//
// It is separate from the GORM pool because scs/mysqlstore writes expiry as a
// UTC time and reads it back with UTC_TIMESTAMP(), while the driver converts
// time parameters into the DSN's loc. With loc=Local (as GORM_DATABASE_URL
// uses) every session would expire off by the local UTC offset, so this pool
// forces loc=UTC.
func OpenSessionDB(dsn string) (*sql.DB, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
	}
	cfg.Loc = time.UTC
	cfg.ParseTime = true

	connector, err := mysqldriver.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("create session db connector: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect session database: %w", err)
	}
	return db, nil
}
