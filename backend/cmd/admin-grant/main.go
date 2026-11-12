// Command admin-grant gives a Firebase Auth uid the owner role on one
// facility (admin_users + admin_facility_access). The MVP has no other role.
// Run from backend/ with the target DSN in GORM_DATABASE_URL. backend/.env is
// loaded first if present (already-set variables win; SKIP_DOTENV=1 disables
// it), so with no override the target is the .env DSN:
//
//	GORM_DATABASE_URL='root:test@tcp(127.0.0.1:33306)/lodging_e2e?parseTime=True' \
//	  go run ./cmd/admin-grant --uid <firebase uid> --facility saku-no-ie
//
// Allowed targets follow scripts/assert-safe-db.sh: the compose DB
// 127.0.0.1:${TEST_DB_PORT:-33306} with lodging_test or lodging_e2e, or
// exactly lodging-master-dev on the shared dev server. Anything else is
// refused before connecting. The DSN is never printed.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	sharedAddr = "192.168.3.100:23306"
	sharedDB   = "lodging-master-dev"
)

func main() {
	uid := flag.String("uid", "", "Firebase Auth uid (localId)")
	facility := flag.String("facility", "", "facility code, e.g. saku-no-ie")
	flag.Parse()
	if _, err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "admin-grant:", err)
		os.Exit(1)
	}
	if err := run(os.Getenv("GORM_DATABASE_URL"), strings.TrimSpace(*uid), strings.TrimSpace(*facility), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "admin-grant:", err)
		os.Exit(1)
	}
}

func run(dsn, uid, facility string, out io.Writer) error {
	if uid == "" || facility == "" {
		return errors.New("--uid and --facility are required")
	}
	if len(uid) > 128 {
		return errors.New("--uid is longer than 128 characters")
	}
	if err := allowedTarget(dsn); err != nil {
		return err
	}
	db, err := database.Open(dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		return errors.New("cannot open the database")
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}
	if err := grant(db, uid, facility, time.Now().UTC().Truncate(time.Microsecond)); err != nil {
		return err
	}
	fmt.Fprintf(out, "granted %s on %s\n", services.RoleOwner, facility)
	return nil
}

// allowedTarget mirrors scripts/assert-safe-db.sh for an additive write.
func allowedTarget(dsn string) error {
	if dsn == "" {
		return errors.New("GORM_DATABASE_URL is required")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return errors.New("refusing: GORM_DATABASE_URL is not a valid go-sql-driver DSN")
	}
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = "33306"
	}
	switch {
	case cfg.Net == "tcp" && cfg.Addr == "127.0.0.1:"+port && (cfg.DBName == "lodging_test" || cfg.DBName == "lodging_e2e"):
		return nil
	case cfg.Net == "tcp" && cfg.Addr == sharedAddr && cfg.DBName == sharedDB:
		return nil
	}
	return errors.New("refusing: target must be compose 127.0.0.1:" + port + " lodging_test/lodging_e2e or " + sharedAddr + "/" + sharedDB)
}

// grant upserts the admin user and the owner row in one transaction.
func grant(db *gorm.DB, uid, code string, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var f entities.Facility
		if err := tx.Where("code = ?", code).Take(&f).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("unknown facility %q", code)
			}
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&entities.AdminUser{FirebaseUID: uid, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		var u entities.AdminUser
		if err := tx.Where("firebase_uid = ?", uid).Take(&u).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"role"})}).
			Create(&entities.AdminFacilityAccess{AdminUserID: u.ID, FacilityID: f.ID, Role: services.RoleOwner, CreatedAt: now}).Error
	})
}
