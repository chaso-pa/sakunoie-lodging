package entities

import "time"

type AdminUser struct {
	ID          uint64 `gorm:"primaryKey"`
	FirebaseUID string `gorm:"column:firebase_uid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (AdminUser) TableName() string { return "admin_users" }
