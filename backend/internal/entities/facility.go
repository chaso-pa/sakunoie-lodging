package entities

import "time"

type Facility struct {
	ID        uint64 `gorm:"primaryKey"`
	Code      string
	Name      string
	Timezone  string
	MaxGuests uint32
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Facility) TableName() string { return "facilities" }
