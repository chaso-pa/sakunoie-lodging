package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxCaptionLength = 2000
	// MaxPostPhotos is how many photos one post can carry.
	MaxPostPhotos    = 5
	maxTags          = 10
	maxTagLength     = 50
	maxWeatherLength = 20
	// ownListLimit bounds the bookmarked and own post lists, which the API
	// returns without paging.
	ownListLimit = 200
	// rankingTitleLength is how much of a caption stands in for a title,
	// as in the prototype (body.slice(0, 28)).
	rankingTitleLength = 28
	// lodgingCategory is left out of the spot ranking, as in the prototype.
	lodgingCategory = "宿泊"
	// cleanupTimeout bounds best-effort deletes of stored images.
	cleanupTimeout = 10 * time.Second
)

// japanTime is the zone "which month was this posted in" is answered in.
// The lodging and its guests are in Japan, whatever zone the server runs in.
var japanTime = time.FixedZone("Asia/Tokyo", 9*60*60)

// postAreas are the regions a post without a spot can be filed under.
var postAreas = map[string]bool{"東信": true, "北信": true, "中信": true, "南信": true}

type PostService struct {
	db     *gorm.DB
	images storage.ImageUploader
	places *PlaceService
}

func NewPostService(db *gorm.DB, images storage.ImageUploader) *PostService {
	return &PostService{db: db, images: images, places: NewPlaceService(db, nil)}
}

// WithPlaces sets the service that shows Google spots, whose names are not
// stored. Without it they are returned with details_available false.
func (s *PostService) WithPlaces(places *PlaceService) *PostService {
	if places != nil {
		s.places = places
	}
	return s
}

// postAreaSQL is the region a post is filed under: its spot's, or the one
// chosen in the form when there is no spot or the spot has no region (a
// member-registered place outside Nagano, or a Google place).
const postAreaSQL = "COALESCE(NULLIF(spots.area, ''), posts.area)"

// PostFilter narrows the feed. Empty fields do not filter.
type PostFilter struct {
	Area    string
	Weather string
	Month   string // YYYY-MM
	Query   string
	Sort    string // "latest" or "popular"
	Limit   int
	Offset  int
}

// List returns one page of the feed as viewerID sees it, and the number of
// posts matching the filter.
func (s *PostService) List(ctx context.Context, viewerID string, f PostFilter) ([]models.PostResponse, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.Post{}).
		Joins("LEFT JOIN spots ON spots.id = posts.spot_id").
		Joins("JOIN members ON members.id = posts.member_id")
	if f.Area != "" {
		q = q.Where(postAreaSQL+" = ?", f.Area)
	}
	if f.Weather != "" {
		q = q.Where("posts.weather = ?", f.Weather)
	}
	if f.Month != "" {
		month, err := time.ParseInLocation("2006-01", f.Month, japanTime)
		if err != nil {
			return nil, 0, invalid("month must be in YYYY-MM format")
		}
		next := month.AddDate(0, 1, 0)
		// visited_at is a calendar date and is compared as one. A post
		// without it is filed under the month it was posted in, which is
		// read in Japan time: created_at is an instant, and the bounds are
		// passed as instants so the driver converts them the same way it
		// converted created_at when the row was written.
		q = q.Where(`(posts.visited_at IS NOT NULL AND posts.visited_at >= ? AND posts.visited_at < ?)
			OR (posts.visited_at IS NULL AND posts.created_at >= ? AND posts.created_at < ?)`,
			month.Format(models.DateLayout), next.Format(models.DateLayout), month, next)
	}
	if query := strings.TrimSpace(f.Query); query != "" {
		like := containsPattern(query)
		q = q.Where(`(posts.caption LIKE ? OR spots.name LIKE ? OR members.nickname LIKE ? OR posts.weather LIKE ?
			OR `+postAreaSQL+` LIKE ?
			OR EXISTS (SELECT 1 FROM post_tags WHERE post_tags.post_id = posts.id AND post_tags.tag LIKE ?))`,
			like, like, like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	if f.Sort == "popular" {
		q = q.Order("(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) DESC")
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	var ids []string
	if err := q.Order("posts.created_at DESC, posts.id").Limit(limit).Offset(max(f.Offset, 0)).
		Pluck("posts.id", &ids).Error; err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	posts, err := s.hydrate(ctx, viewerID, ids)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// Get returns one post as viewerID sees it.
func (s *PostService) Get(ctx context.Context, viewerID, id string) (*models.PostResponse, error) {
	posts, err := s.hydrate(ctx, viewerID, []string{id})
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, ErrPostNotFound
	}
	return &posts[0], nil
}

// Mine returns memberID's own posts, newest first.
func (s *PostService) Mine(ctx context.Context, memberID string) ([]models.PostResponse, error) {
	var ids []string
	if err := s.db.WithContext(ctx).Model(&models.Post{}).
		Where("member_id = ?", memberID).
		Order("created_at DESC, id").Limit(ownListLimit).
		Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list own posts: %w", err)
	}
	return s.hydrate(ctx, memberID, ids)
}

// Bookmarked returns the posts memberID saved, most recently saved first.
func (s *PostService) Bookmarked(ctx context.Context, memberID string) ([]models.PostResponse, error) {
	var ids []string
	if err := s.db.WithContext(ctx).Model(&models.Bookmark{}).
		Where("member_id = ?", memberID).
		Order("created_at DESC, id").Limit(ownListLimit).
		Pluck("post_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	return s.hydrate(ctx, memberID, ids)
}

// hydrate loads the posts with the given ids, in that order, and fills in
// the like count and the viewer's own like and bookmark.
func (s *PostService) hydrate(ctx context.Context, viewerID string, ids []string) ([]models.PostResponse, error) {
	out := []models.PostResponse{}
	if len(ids) == 0 {
		return out, nil
	}
	db := s.db.WithContext(ctx)

	var rows []models.Post
	if err := db.Preload("Member").Preload("Spot").
		Preload("Media", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order, id") }).
		Preload("Tags", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load posts: %w", err)
	}
	byID := make(map[string]*models.Post, len(rows))
	for i := range rows {
		byID[rows[i].ID] = &rows[i]
	}

	likes, err := likeCounts(db, ids)
	if err != nil {
		return nil, err
	}
	spots, err := s.places.Present(ctx, viewerID, postSpots(rows))
	if err != nil {
		return nil, err
	}
	liked, err := memberPostSet(db, &models.Like{}, viewerID, ids)
	if err != nil {
		return nil, err
	}
	bookmarked, err := memberPostSet(db, &models.Bookmark{}, viewerID, ids)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		p, ok := byID[id]
		if !ok {
			continue
		}
		resp := models.PostResponse{
			ID:           p.ID,
			Member:       models.NewMemberSummary(&p.Member),
			Caption:      p.Caption,
			Media:        mediaResponses(p.Media),
			Language:     p.Language,
			Area:         p.Area,
			Weather:      p.Weather,
			Temperature:  p.Temperature,
			VisitedAt:    (*string)(p.VisitedAt),
			Tags:         make([]string, len(p.Tags)),
			LikeCount:    likes[p.ID],
			IsLiked:      liked[p.ID],
			IsBookmarked: bookmarked[p.ID],
			IsMine:       viewerID != "" && p.MemberID == viewerID,
			IsSample:     p.IsSample,
			CreatedAt:    p.CreatedAt,
		}
		for i, t := range p.Tags {
			resp.Tags[i] = t.Tag
		}
		if p.Spot != nil {
			spot := spots[p.Spot.ID]
			resp.Spot = models.NewPostSpotResponse(&spot)
			if p.Spot.Area != "" {
				area := p.Spot.Area
				resp.Area = &area
			}
		}
		out = append(out, resp)
	}
	return out, nil
}

// postSpots collects the spots the posts point at.
func postSpots(posts []models.Post) []*models.Spot {
	var spots []*models.Spot
	for i := range posts {
		if posts[i].Spot != nil {
			spots = append(spots, posts[i].Spot)
		}
	}
	return spots
}

func mediaResponses(media []models.PostMedia) []models.MediaResponse {
	out := make([]models.MediaResponse, len(media))
	for i, m := range media {
		out[i] = models.MediaResponse{URL: m.MediaURL, Type: m.MediaType}
	}
	return out
}

// likeCounts returns the number of likes of each post that has any.
func likeCounts(db *gorm.DB, postIDs []string) (map[string]int, error) {
	counts := map[string]int{}
	if len(postIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		PostID string
		N      int
	}
	if err := db.Model(&models.Like{}).Select("post_id, COUNT(*) AS n").
		Where("post_id IN ?", postIDs).Group("post_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}
	for _, r := range rows {
		counts[r.PostID] = r.N
	}
	return counts, nil
}

// memberPostSet returns which of postIDs memberID has a row for in model's
// table (likes or bookmarks).
func memberPostSet(db *gorm.DB, model any, memberID string, postIDs []string) (map[string]bool, error) {
	set := map[string]bool{}
	if memberID == "" || len(postIDs) == 0 {
		return set, nil
	}
	var ids []string
	if err := db.Model(model).Where("member_id = ? AND post_id IN ?", memberID, postIDs).
		Pluck("post_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load member reactions: %w", err)
	}
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// CreatePostParams is the post form as the handler read it. Media holds the
// photos in the order the member put them, and may be empty.
type CreatePostParams struct {
	Caption     string
	SpotID      string
	Language    string
	Weather     string
	Temperature string
	VisitedAt   string
	Area        string
	Tags        string
	Media       []io.Reader
}

// ParseTags splits a comma separated tag list ("自然, カフェ、秋旅"),
// dropping blanks, a leading # and duplicates.
func ParseTags(input string) ([]string, error) {
	var tags []string
	seen := map[string]bool{}
	for _, part := range strings.FieldsFunc(input, func(r rune) bool { return r == ',' || r == '、' || r == '，' }) {
		tag := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(part), "#＃"))
		if tag == "" || seen[tag] {
			continue
		}
		if utf8.RuneCountInString(tag) > maxTagLength {
			return nil, invalid("each tag must be at most 50 characters")
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	if len(tags) > maxTags {
		return nil, invalid("at most 10 tags are allowed")
	}
	return tags, nil
}

// Create validates the form, stores the photos and inserts the post with its
// media and tag rows in one transaction.
//
// The photos are uploaded before the transaction, because an invalid image
// must reject the request without leaving a post behind. The upload is all
// or nothing (storage.ImageUploader.UploadImages), and if the insert then
// fails, the stored objects are removed again.
func (s *PostService) Create(ctx context.Context, memberID string, p CreatePostParams) (*models.PostResponse, error) {
	caption := strings.TrimSpace(p.Caption)
	if utf8.RuneCountInString(caption) > maxCaptionLength {
		return nil, invalid("caption must be at most 2000 characters")
	}
	if len(p.Media) > MaxPostPhotos {
		return nil, ErrTooManyPhotos
	}
	if caption == "" && len(p.Media) == 0 {
		return nil, invalid("a caption or a photo is required")
	}
	language, err := ValidateLanguage(p.Language)
	if err != nil {
		return nil, invalid(err.Error())
	}
	tags, err := ParseTags(p.Tags)
	if err != nil {
		return nil, err
	}

	post := models.Post{MemberID: memberID, Caption: caption, Language: language}
	if weather := strings.TrimSpace(p.Weather); weather != "" {
		if utf8.RuneCountInString(weather) > maxWeatherLength {
			return nil, invalid("weather must be at most 20 characters")
		}
		post.Weather = &weather
	}
	if raw := strings.TrimSpace(p.Temperature); raw != "" {
		temp, err := strconv.ParseFloat(raw, 64)
		if err != nil || temp < -60 || temp > 60 {
			return nil, invalid("temperature must be a number between -60 and 60")
		}
		post.Temperature = &temp
	}
	if raw := strings.TrimSpace(p.VisitedAt); raw != "" {
		date, err := models.ParseCalendarDate(raw)
		if err != nil {
			return nil, invalid("visited_at must be a date in YYYY-MM-DD format")
		}
		post.VisitedAt = &date
	}
	if p.SpotID != "" {
		// A spot the member may not see is answered like one that does
		// not exist: they cannot attach, and so publish, a place somebody
		// else typed in and has not shared.
		spot, err := findVisibleSpot(s.db.WithContext(ctx), memberID, p.SpotID)
		if errors.Is(err, ErrSpotNotFound) {
			return nil, invalid("spot_id does not match any spot")
		}
		if err != nil {
			return nil, err
		}
		post.SpotID = &spot.ID
		if spot.Area != "" {
			post.Area = &spot.Area
		}
	}
	// The form's region is used when there is no spot, and when the spot
	// has none of its own.
	if area := strings.TrimSpace(p.Area); area != "" && post.Area == nil {
		if !postAreas[area] {
			return nil, invalid("area must be one of 東信, 北信, 中信, 南信")
		}
		post.Area = &area
	}

	var mediaURLs []string
	if len(p.Media) > 0 {
		mediaURLs, err = s.images.UploadImages(ctx, storage.KindPost, p.Media)
		if err != nil {
			return nil, fmt.Errorf("upload post images: %w", err)
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(&post).Error; err != nil {
			return err
		}
		// sort_order is the position the member gave the photo.
		for i, url := range mediaURLs {
			media := models.PostMedia{PostID: post.ID, MediaURL: url, MediaType: "photo", SortOrder: i}
			if err := tx.Create(&media).Error; err != nil {
				return err
			}
		}
		for _, tag := range tags {
			// Time-ordered ids keep the tags in the order they were typed.
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Create(&models.PostTag{ID: id.String(), PostID: post.ID, Tag: tag}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.removeImages(ctx, mediaURLs)
		return nil, fmt.Errorf("create post: %w", err)
	}
	return s.Get(ctx, memberID, post.ID)
}

// Delete removes memberID's post. Its media, tag, like and bookmark rows go
// with it through the foreign keys' ON DELETE CASCADE, and its stored images
// are deleted afterwards. A post of another member is ErrForbidden.
func (s *PostService) Delete(ctx context.Context, memberID, id string) error {
	var urls []string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post models.Post
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPostNotFound
			}
			return err
		}
		if post.MemberID != memberID {
			return ErrForbidden
		}
		if err := tx.Model(&models.PostMedia{}).Where("post_id = ?", id).Pluck("media_url", &urls).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND member_id = ?", id, memberID).Delete(&models.Post{}).Error
	})
	if err != nil {
		if errors.Is(err, ErrPostNotFound) || errors.Is(err, ErrForbidden) {
			return err
		}
		return fmt.Errorf("delete post: %w", err)
	}
	s.removeImages(ctx, urls)
	return nil
}

// removeImages deletes stored images that no row refers to any more. It
// outlives a cancelled request, and a failure is only logged: the object is
// then an orphan in the bucket, which costs space but breaks nothing.
func (s *PostService) removeImages(ctx context.Context, urls []string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	for _, url := range urls {
		if url == "" {
			continue
		}
		if err := s.images.DeleteImage(ctx, url); err != nil {
			slog.WarnContext(ctx, "stored image was not deleted", "url", url, "error", err)
		}
	}
}

// lockPost takes the row lock of the post for the rest of the transaction,
// or reports ErrPostNotFound.
//
// Every like and bookmark toggle of a post goes through this lock, so the
// toggles of one post run one after another. That is what makes them
// deadlock-free: the statements that follow lock index records and gaps in
// likes or bookmarks, and without a common first lock two transactions can
// take those in opposite orders (members liking the same post at once, or
// one member pressing the button twice). The transaction is three short
// statements, so the wait is negligible, and deleting a post takes the same
// lock first.
func (s *PostService) lockPost(tx *gorm.DB, id string) error {
	var ids []string
	if err := tx.Model(&models.Post{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).Limit(1).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return ErrPostNotFound
	}
	return nil
}

// ToggleLike flips memberID's like on the post and returns the new state
// and like count. The (member_id, post_id) unique key keeps a member from
// ever holding two likes on one post, whatever the timing of requests.
func (s *PostService) ToggleLike(ctx context.Context, memberID, postID string) (bool, int, error) {
	var liked bool
	var count int64
	err := retryOnLockConflict(ctx, func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var err error
			liked, err = s.toggle(tx, &models.Like{}, &models.Like{MemberID: memberID, PostID: postID}, memberID, postID)
			if err != nil {
				return err
			}
			return tx.Model(&models.Like{}).Where("post_id = ?", postID).Count(&count).Error
		})
	})
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return false, 0, err
		}
		return false, 0, fmt.Errorf("toggle like: %w", err)
	}
	return liked, int(count), nil
}

// ToggleBookmark flips memberID's bookmark on the post.
func (s *PostService) ToggleBookmark(ctx context.Context, memberID, postID string) (bool, error) {
	var bookmarked bool
	err := retryOnLockConflict(ctx, func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var err error
			bookmarked, err = s.toggle(tx, &models.Bookmark{}, &models.Bookmark{MemberID: memberID, PostID: postID}, memberID, postID)
			return err
		})
	})
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return false, err
		}
		return false, fmt.Errorf("toggle bookmark: %w", err)
	}
	return bookmarked, nil
}

// toggle inserts row (the member's like or bookmark of the post) when there
// is none, and deletes the existing one otherwise. It reports whether the
// row exists afterwards.
//
// It holds the post's row lock (lockPost), and inserts before it deletes:
// deleting first would, when there is no row, leave a gap lock on the unique
// (member_id, post_id) index, which is what made two members liking the
// same post at once deadlock.
//
// "Inserted" is read from RowsAffected: the statement is
// INSERT ... ON DUPLICATE KEY UPDATE id = id, for which MySQL reports 1 row
// when it inserted and 0 when the row was already there and left unchanged
// (the driver does not set CLIENT_FOUND_ROWS).
// TestToggleInsertDetectionIntegration pins that down against the server.
func (s *PostService) toggle(tx *gorm.DB, model, row any, memberID, postID string) (bool, error) {
	if err := s.lockPost(tx, postID); err != nil {
		return false, err
	}
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}
	if err := tx.Where("member_id = ? AND post_id = ?", memberID, postID).Delete(model).Error; err != nil {
		return false, err
	}
	return false, nil
}

// SpotScore is the prototype's popularity formula for a spot
// (docs/pages/sns.html, rankingView): visits + likes + saves*3.
func SpotScore(visits, likes, saves int) int {
	return visits + likes + saves*3
}

// RankingTitle is the first line of a caption cut to 28 characters, which
// is what the prototype shows as a post's title.
func RankingTitle(caption string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(caption), "\n")
	runes := []rune(strings.TrimSpace(line))
	if len(runes) > rankingTitleLength {
		runes = runes[:rankingTitleLength]
	}
	return string(runes)
}

// SpotRanking orders the spots other than the lodging itself by SpotScore.
//
// Visits are the footprints recorded at the spot with is_public set. Private
// footprints are not counted: with few members, a score moving by one would
// tell everyone that somebody was there.
//
// Curated spots are always candidates. A member-registered spot becomes
// one with its first post or public footprint: a place that was picked in
// the composer and never posted about is not a ranking entry, however many
// of those accumulate. A hand-entered spot needs a post, not just its
// creator's public footprint: until then nobody else may see it
// (visibleSpots). Among equal scores curated spots come first.
func (s *PostService) SpotRanking(ctx context.Context, viewerID string, limit int) ([]models.RankingItem, error) {
	var rows []struct {
		models.Spot
		PostCount  int
		LikeCount  int
		SaveCount  int
		VisitCount int
	}
	err := s.db.WithContext(ctx).Table("spots").Select(`spots.*,
		(SELECT COUNT(*) FROM posts WHERE posts.spot_id = spots.id) AS post_count,
		(SELECT COUNT(*) FROM likes JOIN posts ON posts.id = likes.post_id WHERE posts.spot_id = spots.id) AS like_count,
		(SELECT COUNT(*) FROM bookmarks JOIN posts ON posts.id = bookmarks.post_id WHERE posts.spot_id = spots.id) AS save_count,
		(SELECT COUNT(*) FROM gps_visits WHERE gps_visits.spot_id = spots.id AND gps_visits.is_public = TRUE) AS visit_count`).
		Where("spots.category <> ?", lodgingCategory).
		// The same list for every viewer: a hand-entered spot is ranked
		// only once a post has made it visible to all.
		Scopes(visibleSpots("")).
		Where(`spots.source = ?
			OR EXISTS (SELECT 1 FROM posts WHERE posts.spot_id = spots.id)
			OR EXISTS (SELECT 1 FROM gps_visits WHERE gps_visits.spot_id = spots.id AND gps_visits.is_public = TRUE)`, models.SpotSourceSeed).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("rank spots: %w", err)
	}

	// Order by what the rows hold, so that only the spots that make the
	// list are looked up at Google.
	score := func(i int) int { return SpotScore(rows[i].VisitCount, rows[i].LikeCount, rows[i].SaveCount) }
	order := make([]int, len(rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		i, j := order[a], order[b]
		if score(i) != score(j) {
			return score(i) > score(j)
		}
		if seedI, seedJ := rows[i].Source == models.SpotSourceSeed, rows[j].Source == models.SpotSourceSeed; seedI != seedJ {
			return seedI
		}
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].ID < rows[j].ID
	})
	if limit > 0 && len(order) > limit {
		order = order[:limit]
	}
	shown := make([]*models.Spot, len(order))
	for n, i := range order {
		shown[n] = &rows[i].Spot
	}
	views, err := s.places.Present(ctx, viewerID, shown)
	if err != nil {
		return nil, err
	}

	items := make([]models.RankingItem, len(order))
	for n, i := range order {
		r := rows[i]
		id := r.ID
		view := views[r.ID]
		items[n] = models.RankingItem{
			ID:        r.ID,
			Name:      view.Name,
			Score:     score(i),
			Detail:    joinNonEmpty(" · ", view.Area, view.Category),
			Emoji:     view.Emoji,
			Area:      view.Area,
			PostCount: r.PostCount,
			LikeCount: r.LikeCount,
			SpotID:    &id,
			IsSample:  r.IsSample,
		}
	}
	return numbered(items, limit), nil
}

func joinNonEmpty(separator string, parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, separator)
}

// PostRanking orders the posts by like count, newest first among equals.
func (s *PostService) PostRanking(ctx context.Context, viewerID string, limit int) ([]models.RankingItem, error) {
	var ranked []struct {
		ID        string
		LikeCount int
	}
	err := s.db.WithContext(ctx).Table("posts").
		Select("posts.id, COUNT(likes.id) AS like_count").
		Joins("LEFT JOIN likes ON likes.post_id = posts.id").
		Group("posts.id").
		Order("like_count DESC, posts.created_at DESC, posts.id").
		Limit(limit).Scan(&ranked).Error
	if err != nil {
		return nil, fmt.Errorf("rank posts: %w", err)
	}
	if len(ranked) == 0 {
		return []models.RankingItem{}, nil
	}

	ids := make([]string, len(ranked))
	for i, r := range ranked {
		ids[i] = r.ID
	}
	var posts []models.Post
	if err := s.db.WithContext(ctx).Preload("Member").Preload("Spot").Where("id IN ?", ids).Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("load ranked posts: %w", err)
	}
	byID := make(map[string]*models.Post, len(posts))
	for i := range posts {
		byID[posts[i].ID] = &posts[i]
	}

	spots, err := s.places.Present(ctx, viewerID, postSpots(posts))
	if err != nil {
		return nil, err
	}

	items := make([]models.RankingItem, 0, len(ranked))
	for _, r := range ranked {
		p, ok := byID[r.ID]
		if !ok {
			continue
		}
		item := models.RankingItem{
			ID:        p.ID,
			Name:      RankingTitle(p.Caption),
			Score:     r.LikeCount,
			Detail:    p.Member.Nickname,
			LikeCount: r.LikeCount,
			IsSample:  p.IsSample,
		}
		if p.Area != nil {
			item.Area = *p.Area
		}
		if p.Spot != nil {
			spot := spots[p.Spot.ID]
			item.Detail = joinNonEmpty(" · ", spot.Name, p.Member.Nickname)
			item.Emoji = spot.Emoji
			if spot.Area != "" {
				item.Area = spot.Area
			}
			item.SpotID = &p.Spot.ID
			if item.Name == "" {
				item.Name = spot.Name
			}
		}
		items = append(items, item)
	}
	return numbered(items, limit), nil
}

func numbered(items []models.RankingItem, limit int) []models.RankingItem {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	for i := range items {
		items[i].Rank = i + 1
	}
	return items
}
