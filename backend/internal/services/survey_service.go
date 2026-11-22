package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"gorm.io/gorm"
)

// Survey question types (surveydef/*.json).
const (
	SurveyRating       = "rating_1_5"
	SurveySingleChoice = "single_choice"
	SurveyMultiChoice  = "multi_choice"
	SurveyText         = "text"
)

//go:embed surveydef/v1.json
var surveyDefV1JSON []byte

// SurveyQuestion is one question of a versioned survey definition. Labels
// and translations live in frontend/content/surveys/<version>.json.
type SurveyQuestion struct {
	Code      string   `json:"code"`
	Type      string   `json:"type"`
	Required  bool     `json:"required"`
	Options   []string `json:"options"`
	MaxLength int      `json:"maxLength"`
}

type SurveyDefinition struct {
	Version   string           `json:"version"`
	Questions []SurveyQuestion `json:"questions"`
}

func (d *SurveyDefinition) question(code string) (*SurveyQuestion, bool) {
	for i := range d.Questions {
		if d.Questions[i].Code == code {
			return &d.Questions[i], true
		}
	}
	return nil, false
}

// ParseSurveyDefinition decodes and checks a definition.
func ParseSurveyDefinition(b []byte) (SurveyDefinition, error) {
	var d SurveyDefinition
	if err := json.Unmarshal(b, &d); err != nil {
		return d, err
	}
	if d.Version == "" || len(d.Questions) == 0 {
		return d, fmt.Errorf("survey definition: version and questions are required")
	}
	seen := map[string]bool{}
	for _, q := range d.Questions {
		if q.Code == "" || seen[q.Code] {
			return d, fmt.Errorf("survey definition: empty or duplicate code %q", q.Code)
		}
		seen[q.Code] = true
		switch q.Type {
		case SurveyRating:
		case SurveySingleChoice, SurveyMultiChoice:
			if len(q.Options) == 0 || len(slices.Compact(slices.Sorted(slices.Values(q.Options)))) != len(q.Options) {
				return d, fmt.Errorf("survey definition: %s needs distinct options", q.Code)
			}
		case SurveyText:
			if q.MaxLength < 1 {
				return d, fmt.Errorf("survey definition: %s needs maxLength", q.Code)
			}
		default:
			return d, fmt.Errorf("survey definition: %s has unknown type %q", q.Code, q.Type)
		}
	}
	return d, nil
}

// CurrentSurveyDefinition returns the embedded definition (v1).
func CurrentSurveyDefinition() SurveyDefinition {
	d, err := ParseSurveyDefinition(surveyDefV1JSON)
	if err != nil {
		panic("services: invalid embedded surveydef/v1.json: " + err.Error())
	}
	return d
}

// ValidateSurveyAnswers checks answers against the definition and returns
// the normalized answers to store (ratings as integers, text trimmed, empty
// answers dropped). Error locations never echo submitted keys or values.
func ValidateSurveyAnswers(def *SurveyDefinition, answers map[string]any) (map[string]any, error) {
	out := map[string]any{}
	var errs []FieldError
	for _, code := range slices.Sorted(maps.Keys(answers)) {
		v := answers[code]
		q, ok := def.question(code)
		if !ok {
			errs = append(errs, FieldError{"body.answers", "unknown question code"})
			continue
		}
		loc := "body.answers." + q.Code
		if v == nil {
			continue
		}
		switch q.Type {
		case SurveyRating:
			n, ok := v.(float64)
			if !ok || n != math.Trunc(n) || n < 1 || n > 5 {
				errs = append(errs, FieldError{loc, "must be an integer from 1 to 5"})
				continue
			}
			out[q.Code] = int(n)
		case SurveySingleChoice:
			s, ok := v.(string)
			if !ok || !slices.Contains(q.Options, s) {
				errs = append(errs, FieldError{loc, "must be one of the question's options"})
				continue
			}
			out[q.Code] = s
		case SurveyMultiChoice:
			arr, ok := v.([]any)
			if !ok {
				errs = append(errs, FieldError{loc, "must be an array of options"})
				continue
			}
			picked := make([]string, 0, len(arr))
			valid := true
			for _, item := range arr {
				s, ok := item.(string)
				if !ok || !slices.Contains(q.Options, s) || slices.Contains(picked, s) {
					valid = false
					break
				}
				picked = append(picked, s)
			}
			if !valid {
				errs = append(errs, FieldError{loc, "must contain distinct options of the question"})
				continue
			}
			if len(picked) > 0 {
				out[q.Code] = picked
			}
		case SurveyText:
			s, ok := v.(string)
			if !ok {
				errs = append(errs, FieldError{loc, "must be a string"})
				continue
			}
			s = strings.TrimSpace(s)
			if utf8.RuneCountInString(s) > q.MaxLength {
				errs = append(errs, FieldError{loc, fmt.Sprintf("must be at most %d characters", q.MaxLength)})
				continue
			}
			if s != "" {
				out[q.Code] = s
			}
		}
	}
	for _, q := range def.Questions {
		if _, ok := out[q.Code]; q.Required && !ok && !hasFieldError(errs, "body.answers."+q.Code) {
			errs = append(errs, FieldError{"body.answers." + q.Code, "is required"})
		}
	}
	if len(errs) > 0 {
		return nil, newDomainError(ErrValidation, errs...)
	}
	return out, nil
}

func hasFieldError(errs []FieldError, loc string) bool {
	return slices.ContainsFunc(errs, func(e FieldError) bool { return e.Location == loc })
}

// SurveyService stores anonymous survey responses and builds aggregate-only
// summaries for admins (contract §3.7).
type SurveyService struct {
	db           *gorm.DB
	clock        Clock
	def          SurveyDefinition
	minResponses int
}

func NewSurveyService(db *gorm.DB, clock Clock, minResponses int) *SurveyService {
	if minResponses < 1 {
		minResponses = 1
	}
	return &SurveyService{db: db, clock: clock, def: CurrentSurveyDefinition(), minResponses: minResponses}
}

// utcDay truncates t to 00:00:00 UTC of its UTC calendar day.
func utcDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// Submit stores one anonymous response. A filled honeypot is accepted and
// discarded. Both submitted_at and created_at are truncated to the UTC day,
// so stored rows cannot be matched to a stay by submission time.
func (s *SurveyService) Submit(ctx context.Context, in *models.SubmitSurveyBody) error {
	if strings.TrimSpace(in.Website) != "" {
		return nil
	}
	if in.SurveyVersion != s.def.Version {
		return newDomainError(ErrValidation, FieldError{"body.survey_version", "unsupported survey version"})
	}
	db := s.db.WithContext(ctx)
	f, err := loadActiveFacility(db, in.FacilityCode)
	if err != nil {
		return err
	}
	answers, err := ValidateSurveyAnswers(&s.def, in.Answers)
	if err != nil {
		return err
	}
	js, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	day := utcDay(s.clock.Now())
	return db.Create(&entities.SurveyResponse{
		FacilityID:    f.ID,
		SurveyVersion: s.def.Version,
		AnswersJSON:   string(js),
		SubmittedAt:   day,
		CreatedAt:     day,
	}).Error
}

// SurveySummaryFilter is the admin summary filter. Zero values mean "no filter".
type SurveySummaryFilter struct {
	FacilityID uint64
	From, To   string
}

// Summary aggregates the current survey version over the filter. A
// facility outside the scope is 403. Free-text answers are only counted.
func (s *SurveyService) Summary(ctx context.Context, scope AdminScope, f SurveySummaryFilter) (*models.SurveySummary, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	ids, err := facilitiesFilter(scope, f.FacilityID)
	if err != nil {
		return nil, err
	}
	from, to, err := dateRange(f.From, f.To)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&entities.SurveyResponse{}).
		Where("facility_id IN ? AND survey_version = ?", ids, s.def.Version)
	out := &models.SurveySummary{SurveyVersion: s.def.Version, MinResponses: s.minResponses}
	if from != nil {
		q = q.Where("submitted_at >= ?", from.ToDB())
		v := from.String()
		out.From = &v
	}
	if to != nil {
		q = q.Where("submitted_at < ?", to.ToDB().AddDate(0, 0, 1))
		v := to.String()
		out.To = &v
	}
	var rows []string
	if err := q.Order("id").Pluck("answers_json", &rows).Error; err != nil {
		return nil, err
	}
	out.Total = len(rows)
	out.Questions = SummarizeSurvey(&s.def, rows, s.minResponses)
	return out, nil
}

// SummarizeSurvey counts answers per question. A question answered by fewer
// than minResponses responses has its mean and counts suppressed (null).
// Invalid stored values are ignored.
func SummarizeSurvey(def *SurveyDefinition, rows []string, minResponses int) []models.SurveyQuestionSummary {
	type acc struct {
		answered int
		sum      int
		counts   map[string]int
	}
	accs := make([]acc, len(def.Questions))
	for i := range accs {
		accs[i].counts = map[string]int{}
	}
	for _, raw := range rows {
		var answers map[string]any
		if json.Unmarshal([]byte(raw), &answers) != nil {
			continue
		}
		for i, q := range def.Questions {
			v, ok := answers[q.Code]
			if !ok {
				continue
			}
			a := &accs[i]
			switch q.Type {
			case SurveyRating:
				n, ok := v.(float64)
				if !ok || n != math.Trunc(n) || n < 1 || n > 5 {
					continue
				}
				a.answered++
				a.sum += int(n)
				a.counts[strconv.Itoa(int(n))]++
			case SurveySingleChoice:
				if s, ok := v.(string); ok && slices.Contains(q.Options, s) {
					a.answered++
					a.counts[s]++
				}
			case SurveyMultiChoice:
				arr, ok := v.([]any)
				if !ok {
					continue
				}
				counted := false
				for _, item := range arr {
					if s, ok := item.(string); ok && slices.Contains(q.Options, s) {
						a.counts[s]++
						counted = true
					}
				}
				if counted {
					a.answered++
				}
			case SurveyText:
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					a.answered++
				}
			}
		}
	}

	out := make([]models.SurveyQuestionSummary, len(def.Questions))
	for i, q := range def.Questions {
		a := accs[i]
		qs := models.SurveyQuestionSummary{Code: q.Code, Type: q.Type, Answered: a.answered}
		if a.answered < minResponses {
			qs.Suppressed = true
			out[i] = qs
			continue
		}
		values := q.Options
		if q.Type == SurveyRating {
			values = []string{"1", "2", "3", "4", "5"}
			mean := math.Round(float64(a.sum)/float64(a.answered)*100) / 100
			qs.Mean = &mean
		}
		if q.Type != SurveyText {
			qs.Counts = make([]models.SurveyCount, len(values))
			for j, v := range values {
				qs.Counts[j] = models.SurveyCount{Value: v, Count: a.counts[v]}
			}
		}
		out[i] = qs
	}
	return out
}
