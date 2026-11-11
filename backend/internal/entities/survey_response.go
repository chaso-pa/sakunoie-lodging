package entities

import "time"

type SurveyResponse struct {
	ID            uint64 `gorm:"primaryKey"`
	FacilityID    uint64
	SurveyVersion string
	AnswersJSON   string `gorm:"column:answers_json"`
	SubmittedAt   time.Time
	CreatedAt     time.Time
}

func (SurveyResponse) TableName() string { return "survey_responses" }
