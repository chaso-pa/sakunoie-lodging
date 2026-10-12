package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultVisitEmoji = "📍"
	defaultVisitArea  = "未設定"
	maxVisitNote      = 2000
	visitListLimit    = 500
)

type GpsVisitService struct {
	db *gorm.DB
}

func NewGpsVisitService(db *gorm.DB) *GpsVisitService {
	return &GpsVisitService{db: db}
}

// visitResponse is the only place a footprint is turned into what a member
// sees, and so the only place its coordinates can leave the server.
//
//   - The owner always gets their own coordinates.
//   - Anyone else gets them only when the visit is public and its owner
//     chose to publish the exact position. Otherwise both are null.
//
// Callers must not hand a non-public visit of another member to this
// function at all; ListPublic filters those out in SQL. The check here is a
// second line of defence for that case.
func visitResponse(v *models.GpsVisit, viewerID string) models.GpsVisitResponse {
	mine := viewerID != "" && v.MemberID == viewerID
	resp := models.GpsVisitResponse{
		ID:          v.ID,
		Member:      models.NewMemberSummary(&v.Member),
		IsMine:      mine,
		SpotID:      v.SpotID,
		PlaceName:   v.PlaceName,
		Emoji:       v.Emoji,
		Area:        v.Area,
		VisitedDate: v.VisitedDate.Format(models.DateLayout),
		Weather:     v.Weather,
		Temperature: v.Temperature,
		Note:        v.Note,
		IsPublic:    v.IsPublic,
		ExactPublic: v.ExactPublic,
		CreatedAt:   v.CreatedAt,
	}
	if mine || (v.IsPublic && v.ExactPublic) {
		resp.Latitude = v.Latitude
		resp.Longitude = v.Longitude
	}
	return resp
}

func visitResponses(visits []models.GpsVisit, viewerID string) []models.GpsVisitResponse {
	out := make([]models.GpsVisitResponse, len(visits))
	for i := range visits {
		out[i] = visitResponse(&visits[i], viewerID)
	}
	return out
}

// ListMine returns every footprint of memberID, public or not.
func (s *GpsVisitService) ListMine(ctx context.Context, memberID string) ([]models.GpsVisitResponse, error) {
	var visits []models.GpsVisit
	if err := s.db.WithContext(ctx).Preload("Member").
		Where("member_id = ?", memberID).
		Order("visited_date DESC, created_at DESC, id").Limit(visitListLimit).
		Find(&visits).Error; err != nil {
		return nil, fmt.Errorf("list own visits: %w", err)
	}
	return visitResponses(visits, memberID), nil
}

// ListPublic returns the footprints every member may see: those with
// is_public set, including the viewer's own public ones. Non-public visits
// are never loaded.
func (s *GpsVisitService) ListPublic(ctx context.Context, viewerID string) ([]models.GpsVisitResponse, error) {
	var visits []models.GpsVisit
	if err := s.db.WithContext(ctx).Preload("Member").
		Where("is_public = ?", true).
		Order("visited_date DESC, created_at DESC, id").Limit(visitListLimit).
		Find(&visits).Error; err != nil {
		return nil, fmt.Errorf("list public visits: %w", err)
	}
	return visitResponses(visits, viewerID), nil
}

// CreateGpsVisitParams is a new footprint. IsPublic and ExactPublic both
// default to false when nil: a footprint is private, and its exact position
// hidden, unless the member turns each on. A client that forgets a field
// must not publish a visit.
type CreateGpsVisitParams struct {
	SpotID      string
	PlaceName   string
	Emoji       string
	Area        string
	VisitedDate string
	Weather     string
	Temperature *float64
	Note        string
	IsPublic    *bool
	ExactPublic *bool
	Latitude    *float64
	Longitude   *float64
}

func (s *GpsVisitService) Create(ctx context.Context, memberID string, p CreateGpsVisitParams) (*models.GpsVisitResponse, error) {
	date, err := time.ParseInLocation(models.DateLayout, strings.TrimSpace(p.VisitedDate), time.Local)
	if err != nil {
		return nil, invalid("visited_date must be a date in YYYY-MM-DD format")
	}
	if (p.Latitude == nil) != (p.Longitude == nil) {
		return nil, invalid("latitude and longitude must be given together")
	}
	if p.Latitude != nil && (*p.Latitude < -90 || *p.Latitude > 90 || *p.Longitude < -180 || *p.Longitude > 180) {
		return nil, invalid("latitude or longitude is out of range")
	}
	if p.Temperature != nil && (*p.Temperature < -60 || *p.Temperature > 60) {
		return nil, invalid("temperature must be between -60 and 60")
	}

	visit := models.GpsVisit{
		MemberID:    memberID,
		PlaceName:   strings.TrimSpace(p.PlaceName),
		Emoji:       strings.TrimSpace(p.Emoji),
		Area:        strings.TrimSpace(p.Area),
		VisitedDate: date,
		Temperature: p.Temperature,
		IsPublic:    p.IsPublic != nil && *p.IsPublic,
		ExactPublic: p.ExactPublic != nil && *p.ExactPublic,
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
	}
	if weather := strings.TrimSpace(p.Weather); weather != "" {
		if utf8.RuneCountInString(weather) > maxWeatherLength {
			return nil, invalid("weather must be at most 20 characters")
		}
		visit.Weather = &weather
	}
	if note := strings.TrimSpace(p.Note); note != "" {
		if utf8.RuneCountInString(note) > maxVisitNote {
			return nil, invalid("note must be at most 2000 characters")
		}
		visit.Note = &note
	}

	if p.SpotID != "" {
		var spot models.Spot
		if err := s.db.WithContext(ctx).Where("id = ?", p.SpotID).First(&spot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, invalid("spot_id does not match any spot")
			}
			return nil, fmt.Errorf("get spot: %w", err)
		}
		visit.SpotID = &spot.ID
		visit.PlaceName = orDefault(visit.PlaceName, spot.Name)
		visit.Emoji = orDefault(visit.Emoji, spot.Emoji)
		visit.Area = orDefault(visit.Area, spot.Area)
	}
	if visit.PlaceName == "" {
		return nil, invalid("place_name is required when spot_id is not given")
	}
	if utf8.RuneCountInString(visit.PlaceName) > 100 || utf8.RuneCountInString(visit.Emoji) > 10 || utf8.RuneCountInString(visit.Area) > 20 {
		return nil, invalid("place_name, emoji or area is too long")
	}
	visit.Emoji = orDefault(visit.Emoji, defaultVisitEmoji)
	visit.Area = orDefault(visit.Area, defaultVisitArea)

	if err := s.db.WithContext(ctx).Omit(clause.Associations).Create(&visit).Error; err != nil {
		return nil, fmt.Errorf("create visit: %w", err)
	}
	return s.get(ctx, memberID, visit.ID)
}

// owned loads the visit for a write by memberID. A visit of another member
// that the caller could not see anyway is reported as missing, so its
// existence is not revealed; a public one is ErrForbidden.
func (s *GpsVisitService) owned(tx *gorm.DB, memberID, id string) (*models.GpsVisit, error) {
	var visit models.GpsVisit
	if err := tx.Where("id = ?", id).First(&visit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVisitNotFound
		}
		return nil, err
	}
	if visit.MemberID != memberID {
		if !visit.IsPublic {
			return nil, ErrVisitNotFound
		}
		return nil, ErrForbidden
	}
	return &visit, nil
}

func (s *GpsVisitService) get(ctx context.Context, memberID, id string) (*models.GpsVisitResponse, error) {
	var visit models.GpsVisit
	if err := s.db.WithContext(ctx).Preload("Member").Where("id = ? AND member_id = ?", id, memberID).First(&visit).Error; err != nil {
		return nil, fmt.Errorf("reload visit: %w", err)
	}
	resp := visitResponse(&visit, memberID)
	return &resp, nil
}

// Delete removes memberID's footprint.
func (s *GpsVisitService) Delete(ctx context.Context, memberID, id string) error {
	if _, err := s.owned(s.db.WithContext(ctx), memberID, id); err != nil {
		return wrapUnlessKnown("delete visit", err)
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND member_id = ?", id, memberID).Delete(&models.GpsVisit{}).Error; err != nil {
		return fmt.Errorf("delete visit: %w", err)
	}
	return nil
}

// SetVisibility changes who can see memberID's footprint. exactPublic nil
// leaves that setting as it is.
func (s *GpsVisitService) SetVisibility(ctx context.Context, memberID, id string, isPublic bool, exactPublic *bool) (*models.GpsVisitResponse, error) {
	if _, err := s.owned(s.db.WithContext(ctx), memberID, id); err != nil {
		return nil, wrapUnlessKnown("update visit", err)
	}
	updates := map[string]any{"is_public": isPublic}
	if exactPublic != nil {
		updates["exact_public"] = *exactPublic
	}
	if err := s.db.WithContext(ctx).Model(&models.GpsVisit{}).
		Where("id = ? AND member_id = ?", id, memberID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update visit: %w", err)
	}
	return s.get(ctx, memberID, id)
}

func wrapUnlessKnown(op string, err error) error {
	if errors.Is(err, ErrVisitNotFound) || errors.Is(err, ErrForbidden) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
