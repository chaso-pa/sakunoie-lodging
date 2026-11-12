package entities

import "time"

type AdminFacilityAccess struct {
	AdminUserID uint64 `gorm:"primaryKey;autoIncrement:false"`
	FacilityID  uint64 `gorm:"primaryKey;autoIncrement:false"`
	Role        string
	CreatedAt   time.Time
}

func (AdminFacilityAccess) TableName() string { return "admin_facility_access" }
