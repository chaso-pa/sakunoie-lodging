// Package entities holds GORM structs for the tables in prisma/schema.prisma.
//
// Mapping rules (plan Phase 2):
//   - DATETIME(6) fields are time.Time in UTC (internal/database forces loc=UTC).
//   - DATE fields are time.Time at midnight UTC; convert with
//     utils.CivilDateFromDB before any time-zone math or display.
//   - TIME fields are *string "HH:MM:SS"; parseTime does not parse TIME.
//   - JSON columns are LONGTEXT with json_valid() CHECKs: *string / string
//     holding JSON marshalled by the caller, never a GORM JSON type.
//   - DeletedAt is a plain *time.Time, not gorm.DeletedAt, so GORM adds no
//     implicit scope; services apply notDeleted explicitly.
//   - import_batches / import_rows are not modelled (CSV import is out of scope).
package entities
