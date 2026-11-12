package main

import (
	"strings"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

func TestAllowedTarget(t *testing.T) {
	t.Setenv("TEST_DB_PORT", "")
	ok := []string{
		"root:test@tcp(127.0.0.1:33306)/lodging_test?parseTime=True",
		"root:test@tcp(127.0.0.1:33306)/lodging_e2e",
		"dev:x@tcp(192.168.3.100:23306)/lodging-master-dev?charset=utf8mb4&parseTime=True&loc=Local",
	}
	refused := []string{
		"",
		"not a dsn",
		"dev:x@tcp(192.168.3.100:23306)/sakunoie-dev",
		"dev:x@tcp(192.168.3.100:23306)/lodging-manager-dev",
		"dev:x@tcp(192.168.3.100:23306)/",
		"root:test@tcp(127.0.0.1:33306)/lodging_migrate",
		"root:test@tcp(localhost:33306)/lodging_test",
		"root:test@tcp(127.0.0.1:3306)/lodging_test",
		"root:test@tcp(127.0.0.1:3307)/lodging_e2e",
		"root:test@unix(/tmp/mysql.sock)/lodging_test",
	}
	for _, dsn := range ok {
		if err := allowedTarget(dsn); err != nil {
			t.Errorf("refused %q: %v", dsn, err)
		}
	}
	for _, dsn := range refused {
		err := allowedTarget(dsn)
		if err == nil {
			t.Errorf("allowed %q", dsn)
		} else if strings.Contains(err.Error(), "test@") || strings.Contains(err.Error(), "x@") {
			t.Errorf("error echoes the DSN: %v", err)
		}
	}
	if err := run("dev:x@tcp(192.168.3.100:23306)/sakunoie-dev", "uid", "saku-no-ie", nil); err == nil {
		t.Fatal("run did not refuse a foreign shared DB")
	}
}

func TestIntegrationAdminGrant(t *testing.T) {
	db := testutil.RequireDB(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	for range 2 { // idempotent
		if err := grant(db, "test-grant-uid", testutil.FacilitySaku, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := grant(db, "test-grant-uid", "no-such-facility", now); err == nil {
		t.Fatal("unknown facility accepted")
	}
	var rows []entities.AdminFacilityAccess
	db.Find(&rows)
	if len(rows) != 1 || rows[0].Role != services.RoleOwner || testutil.CountRows(t, db, "admin_users") != 1 {
		t.Fatalf("access rows = %+v", rows)
	}
	// A non-owner role is upgraded to owner.
	db.Model(&entities.AdminFacilityAccess{}).Where("admin_user_id = ?", rows[0].AdminUserID).Update("role", "staff")
	if err := grant(db, "test-grant-uid", testutil.FacilitySaku, now); err != nil {
		t.Fatal(err)
	}
	db.Find(&rows)
	if rows[0].Role != services.RoleOwner {
		t.Fatalf("role = %s", rows[0].Role)
	}
	// Authenticate resolves the granted uid.
	auth := services.NewAdminAuthService(db, testutil.FakeVerifier{"t": "test-grant-uid"})
	scope, err := auth.Authenticate(t.Context(), "t")
	if err != nil || len(scope.FacilityIDs) != 1 {
		t.Fatalf("scope = %+v %v", scope, err)
	}
}
