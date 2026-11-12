package entities

import "time"

type Guest struct {
	ID                       uint64 `gorm:"primaryKey"`
	PublicID                 string
	RegistrationID           uint64
	IsRepresentative         bool
	FullName                 string
	NameKana                 *string
	Address                  string
	Phone                    *string
	Email                    *string
	HasJapanAddress          bool
	Nationality              *string
	PassportNumberCiphertext []byte
	PassportNumberLast4      *string `gorm:"column:passport_number_last4"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (Guest) TableName() string { return "guests" }
