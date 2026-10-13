package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"gorm.io/gorm"
)

const (
	spotPostLimit = 20
	// spotReviewLimit bounds the reviews on a spot's page to the newest ones.
	spotReviewLimit = 100
	maxReviewLength = 1000
	// defaultAiGenre matches every AI spot, as in the prototype.
	defaultAiGenre = "穴場スポット"
)

type SpotService struct {
	db     *gorm.DB
	places *PlaceService
}

func NewSpotService(db *gorm.DB) *SpotService {
	return &SpotService{db: db, places: NewPlaceService(db, nil)}
}

// WithPlaces sets the service that shows Google spots, whose names are not
// stored. Without it they are returned with details_available false.
func (s *SpotService) WithPlaces(places *PlaceService) *SpotService {
	if places != nil {
		s.places = places
	}
	return s
}

// visibleSpots keeps the spots memberID may see or use. It is the one rule
// for every place a spot is looked up, by id or by text:
//
//	a spot is visible when it is not hand-entered, or the member entered
//	it, or at least one post points at it.
//
// A hand-entered place is member-written text that may be anything, a home
// address included, so it stays with the member who typed it until they
// (or, once it is visible, anyone) publish it by posting with it. To
// everyone else it does not exist: lookups answer exactly as they do for
// an id that was never issued.
//
// The rule is evaluated on every read and nothing records that a spot
// "was once posted". Deleting the last post that uses a hand-entered spot
// therefore returns it to its creator alone: deleting the post is how a
// member takes back what they published, and the place goes with it.
//
// An empty memberID is nobody: only the first and last conditions can
// hold, which is what viewer-independent lists (the ranking) use.
func visibleSpots(memberID string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(`(spots.source <> ? OR spots.created_by_member_id = ?
			OR EXISTS (SELECT 1 FROM posts WHERE posts.spot_id = spots.id))`, models.SpotSourceManual, memberID)
	}
}

// findVisibleSpot loads the spot with this id if memberID may see it, and
// returns ErrSpotNotFound otherwise, whether the spot is missing or hidden.
func findVisibleSpot(db *gorm.DB, memberID, id string) (*models.Spot, error) {
	var spot models.Spot
	if err := db.Scopes(visibleSpots(memberID)).Where("spots.id = ?", id).First(&spot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSpotNotFound
		}
		return nil, fmt.Errorf("get spot: %w", err)
	}
	return &spot, nil
}

// List returns the curated spots, optionally of one area, with their
// review and post counts.
//
// The endpoint is public, so it lists the operator's own spots (source
// seed) and nothing a member registered. A member-registered spot carries
// member-generated content (a hand-typed name and address, or simply the
// fact that a guest picked that place), which is for guests only; a Google
// spot would also cost a billed lookup for every anonymous visitor. Those
// spots are reached through posts, GET /spots/{id} and place search, all
// of which need a session.
func (s *SpotService) List(ctx context.Context, area string) ([]models.SpotListItem, error) {
	q := s.db.WithContext(ctx).Where("source = ?", models.SpotSourceSeed).Order("created_at, name")
	if area != "" {
		q = q.Where("area = ?", area)
	}
	var spots []models.Spot
	if err := q.Find(&spots).Error; err != nil {
		return nil, fmt.Errorf("list spots: %w", err)
	}

	var reviewStats []struct {
		SpotID string
		N      int
		Avg    float64
	}
	if err := s.db.WithContext(ctx).Model(&models.Review{}).
		Select("spot_id, COUNT(*) AS n, AVG(stars) AS avg").
		Group("spot_id").Scan(&reviewStats).Error; err != nil {
		return nil, fmt.Errorf("count reviews: %w", err)
	}
	var postStats []struct {
		SpotID string
		N      int
	}
	if err := s.db.WithContext(ctx).Model(&models.Post{}).
		Select("spot_id, COUNT(*) AS n").
		Where("spot_id IS NOT NULL").
		Group("spot_id").Scan(&postStats).Error; err != nil {
		return nil, fmt.Errorf("count posts: %w", err)
	}

	items := make([]models.SpotListItem, len(spots))
	index := make(map[string]int, len(spots))
	for i := range spots {
		items[i] = models.SpotListItem{SpotResponse: models.NewSpotResponse(&spots[i])}
		index[spots[i].ID] = i
	}
	for _, r := range reviewStats {
		if i, ok := index[r.SpotID]; ok {
			items[i].ReviewCount = r.N
			items[i].AvgStars = float64(int(r.Avg*10+0.5)) / 10
		}
	}
	for _, p := range postStats {
		if i, ok := index[p.SpotID]; ok {
			items[i].PostCount = p.N
		}
	}
	return items, nil
}

// SpotDetail is a spot with its reviews and latest posts.
type SpotDetail struct {
	Spot    models.SpotResponse
	Reviews []models.ReviewResponse
	Posts   []models.SpotPostResponse
}

// Get returns the spot as viewerID sees it, or ErrSpotNotFound when it is
// not visible to them (visibleSpots).
func (s *SpotService) Get(ctx context.Context, viewerID, id string) (*SpotDetail, error) {
	spot, err := findVisibleSpot(s.db.WithContext(ctx), viewerID, id)
	if err != nil {
		return nil, err
	}
	view, err := s.places.PresentOne(ctx, viewerID, spot)
	if err != nil {
		return nil, err
	}

	var reviews []models.Review
	if err := s.db.WithContext(ctx).Preload("Member").
		Where("spot_id = ?", id).Order("created_at DESC, id").Limit(spotReviewLimit).Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}

	var posts []models.Post
	if err := s.db.WithContext(ctx).Preload("Member").
		Preload("Media", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order, id") }).
		Where("spot_id = ?", id).Order("created_at DESC, id").Limit(spotPostLimit).Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("list spot posts: %w", err)
	}
	ids := make([]string, len(posts))
	for i := range posts {
		ids[i] = posts[i].ID
	}
	likes, err := likeCounts(s.db.WithContext(ctx), ids)
	if err != nil {
		return nil, err
	}

	detail := &SpotDetail{
		Spot:    view,
		Reviews: make([]models.ReviewResponse, len(reviews)),
		Posts:   make([]models.SpotPostResponse, len(posts)),
	}
	for i := range reviews {
		detail.Reviews[i] = models.NewReviewResponse(&reviews[i])
	}
	for i := range posts {
		p := &posts[i]
		detail.Posts[i] = models.SpotPostResponse{
			ID:        p.ID,
			Caption:   p.Caption,
			Media:     mediaResponses(p.Media),
			Member:    models.NewMemberSummary(&p.Member),
			LikeCount: likes[p.ID],
			IsSample:  p.IsSample,
			CreatedAt: p.CreatedAt,
		}
	}
	return detail, nil
}

// CreateReview adds memberID's review to the spot.
func (s *SpotService) CreateReview(ctx context.Context, memberID, spotID string, stars int, text string) (*models.ReviewResponse, error) {
	text = strings.TrimSpace(text)
	if stars < 1 || stars > 5 {
		return nil, invalid("stars must be between 1 and 5")
	}
	if text == "" || utf8.RuneCountInString(text) > maxReviewLength {
		return nil, invalid("text must be 1 to 1000 characters")
	}
	if _, err := findVisibleSpot(s.db.WithContext(ctx), memberID, spotID); err != nil {
		return nil, err
	}

	review := models.Review{MemberID: memberID, SpotID: spotID, Stars: stars, Text: text}
	if err := s.db.WithContext(ctx).Omit("Member").Create(&review).Error; err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}
	if err := s.db.WithContext(ctx).Preload("Member").First(&review, "id = ?", review.ID).Error; err != nil {
		return nil, fmt.Errorf("reload review: %w", err)
	}
	resp := models.NewReviewResponse(&review)
	return &resp, nil
}

// AiSpotFilter mirrors the selectors of the prototype's AI spot panel.
type AiSpotFilter struct {
	Genre     string
	Purpose   string
	Weather   string
	Duration  string
	Transport string
}

// ListAiSpots ports the prototype's recommendation list: filter by genre,
// purpose and weather, relax the filter when nothing matches, order by
// travel time for the chosen transport and keep 2, 3 or 5 by duration.
func (s *SpotService) ListAiSpots(ctx context.Context, f AiSpotFilter) ([]models.AiSpotResponse, error) {
	var rows []models.AiSpot
	if err := s.db.WithContext(ctx).Order("name").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list ai spots: %w", err)
	}
	genre := orDefault(f.Genre, defaultAiGenre)
	purpose := orDefault(f.Purpose, "all")
	weather := orDefault(f.Weather, "any")

	matchGenre := func(r *models.AiSpot) bool {
		return genre == defaultAiGenre || containsString(r.Genres, genre)
	}
	matchPurpose := func(r *models.AiSpot) bool {
		return purpose == "all" || containsString(r.Purposes, purpose)
	}
	matchWeather := func(r *models.AiSpot) bool {
		return weather == "any" || containsString(r.Weathers, weather)
	}
	filters := []func(r *models.AiSpot) bool{
		func(r *models.AiSpot) bool { return matchGenre(r) && matchPurpose(r) && matchWeather(r) },
		func(r *models.AiSpot) bool { return matchPurpose(r) && matchWeather(r) },
		matchPurpose,
	}
	var picked []*models.AiSpot
	for _, keep := range filters {
		for i := range rows {
			if keep(&rows[i]) {
				picked = append(picked, &rows[i])
			}
		}
		if len(picked) > 0 {
			break
		}
	}

	minutes := func(r *models.AiSpot) int {
		if f.Transport == "transit" {
			return r.TransitMin
		}
		return r.DriveMin
	}
	sort.SliceStable(picked, func(i, j int) bool { return minutes(picked[i]) < minutes(picked[j]) })

	limit := 3
	switch f.Duration {
	case "short":
		limit = 2
	case "full":
		limit = 5
	}
	if len(picked) > limit {
		picked = picked[:limit]
	}

	out := make([]models.AiSpotResponse, len(picked))
	for i, r := range picked {
		out[i] = models.AiSpotResponse{
			ID:          r.ID,
			Name:        r.Name,
			Area:        r.Area,
			Kind:        r.Kind,
			DriveMin:    r.DriveMin,
			TransitMin:  r.TransitMin,
			StayMin:     r.StayMin,
			Description: r.Description,
			MapQuery:    r.MapQuery,
			SearchURL:   r.SearchURL,
		}
	}
	return out, nil
}

// containsString reports whether the JSON string array encoded contains want.
func containsString(encoded, want string) bool {
	var values []string
	if err := json.Unmarshal([]byte(encoded), &values); err != nil {
		return false
	}
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
