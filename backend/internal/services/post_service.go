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

// postAreas are the regions a post without a spot can be filed under.
var postAreas = map[string]bool{"東信": true, "北信": true, "中信": true, "南信": true}

type PostService struct {
	db     *gorm.DB
	images storage.ImageUploader
}

func NewPostService(db *gorm.DB, images storage.ImageUploader) *PostService {
	return &PostService{db: db, images: images}
}

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
		q = q.Where("COALESCE(spots.area, posts.area) = ?", f.Area)
	}
	if f.Weather != "" {
		q = q.Where("posts.weather = ?", f.Weather)
	}
	if f.Month != "" {
		q = q.Where("DATE_FORMAT(COALESCE(posts.visited_at, posts.created_at), '%Y-%m') = ?", f.Month)
	}
	if query := strings.TrimSpace(f.Query); query != "" {
		like := containsPattern(query)
		q = q.Where(`(posts.caption LIKE ? OR spots.name LIKE ? OR members.nickname LIKE ? OR posts.weather LIKE ?
			OR COALESCE(spots.area, posts.area) LIKE ?
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
			VisitedAt:    models.FormatDate(p.VisitedAt),
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
			resp.Spot = &models.PostSpotResponse{ID: p.Spot.ID, Name: p.Spot.Name, Emoji: p.Spot.Emoji, Area: p.Spot.Area}
			area := p.Spot.Area
			resp.Area = &area
		}
		out = append(out, resp)
	}
	return out, nil
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

// CreatePostParams is the post form as the handler read it. Media may be nil.
type CreatePostParams struct {
	Caption     string
	SpotID      string
	Language    string
	Weather     string
	Temperature string
	VisitedAt   string
	Area        string
	Tags        string
	Media       io.Reader
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

// Create validates the form, stores the photo and inserts the post with its
// media and tag rows in one transaction.
//
// The photo is uploaded before the transaction, because an invalid image
// must reject the request without leaving a post behind. If the insert then
// fails, the stored object is removed again.
func (s *PostService) Create(ctx context.Context, memberID string, p CreatePostParams) (*models.PostResponse, error) {
	caption := strings.TrimSpace(p.Caption)
	if utf8.RuneCountInString(caption) > maxCaptionLength {
		return nil, invalid("caption must be at most 2000 characters")
	}
	if caption == "" && p.Media == nil {
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
		date, err := time.ParseInLocation(models.DateLayout, raw, time.Local)
		if err != nil {
			return nil, invalid("visited_at must be a date in YYYY-MM-DD format")
		}
		post.VisitedAt = &date
	}
	if p.SpotID != "" {
		var spot models.Spot
		if err := s.db.WithContext(ctx).Where("id = ?", p.SpotID).First(&spot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, invalid("spot_id does not match any spot")
			}
			return nil, fmt.Errorf("get spot: %w", err)
		}
		post.SpotID = &spot.ID
		post.Area = &spot.Area
	} else if area := strings.TrimSpace(p.Area); area != "" {
		if !postAreas[area] {
			return nil, invalid("area must be one of 東信, 北信, 中信, 南信")
		}
		post.Area = &area
	}

	var mediaURL string
	if p.Media != nil {
		mediaURL, err = s.images.UploadImage(ctx, storage.KindPost, p.Media)
		if err != nil {
			return nil, fmt.Errorf("upload post image: %w", err)
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(&post).Error; err != nil {
			return err
		}
		if mediaURL != "" {
			media := models.PostMedia{PostID: post.ID, MediaURL: mediaURL, MediaType: "photo"}
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
		s.removeImages(ctx, []string{mediaURL})
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

func (s *PostService) exists(tx *gorm.DB, id string) error {
	var n int64
	if err := tx.Model(&models.Post{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.toggle(tx, &models.Like{}, &models.Like{MemberID: memberID, PostID: postID}, memberID, postID)
		if err != nil {
			return err
		}
		return tx.Model(&models.Like{}).Where("post_id = ?", postID).Count(&count).Error
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		bookmarked, err = s.toggle(tx, &models.Bookmark{}, &models.Bookmark{MemberID: memberID, PostID: postID}, memberID, postID)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return false, err
		}
		return false, fmt.Errorf("toggle bookmark: %w", err)
	}
	return bookmarked, nil
}

// toggle deletes the member's row for the post when there is one, and
// inserts row otherwise. It reports whether the row exists afterwards.
func (s *PostService) toggle(tx *gorm.DB, model, row any, memberID, postID string) (bool, error) {
	if err := s.exists(tx, postID); err != nil {
		return false, err
	}
	res := tx.Where("member_id = ? AND post_id = ?", memberID, postID).Delete(model)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return false, nil
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
		return false, err
	}
	return true, nil
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
func (s *PostService) SpotRanking(ctx context.Context, limit int) ([]models.RankingItem, error) {
	var rows []struct {
		ID         string
		Name       string
		Emoji      string
		Area       string
		Category   string
		IsSample   bool
		PostCount  int
		LikeCount  int
		SaveCount  int
		VisitCount int
	}
	err := s.db.WithContext(ctx).Table("spots").Select(`spots.id, spots.name, spots.emoji, spots.area, spots.category, spots.is_sample,
		(SELECT COUNT(*) FROM posts WHERE posts.spot_id = spots.id) AS post_count,
		(SELECT COUNT(*) FROM likes JOIN posts ON posts.id = likes.post_id WHERE posts.spot_id = spots.id) AS like_count,
		(SELECT COUNT(*) FROM bookmarks JOIN posts ON posts.id = bookmarks.post_id WHERE posts.spot_id = spots.id) AS save_count,
		(SELECT COUNT(*) FROM gps_visits WHERE gps_visits.spot_id = spots.id AND gps_visits.is_public = TRUE) AS visit_count`).
		Where("spots.category <> ?", lodgingCategory).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("rank spots: %w", err)
	}

	items := make([]models.RankingItem, len(rows))
	for i, r := range rows {
		id := r.ID
		items[i] = models.RankingItem{
			ID:        r.ID,
			Name:      r.Name,
			Score:     SpotScore(r.VisitCount, r.LikeCount, r.SaveCount),
			Detail:    r.Area + " · " + r.Category,
			Emoji:     r.Emoji,
			Area:      r.Area,
			PostCount: r.PostCount,
			LikeCount: r.LikeCount,
			SpotID:    &id,
			IsSample:  r.IsSample,
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].Name < items[j].Name
	})
	return numbered(items, limit), nil
}

// PostRanking orders the posts by like count, newest first among equals.
func (s *PostService) PostRanking(ctx context.Context, limit int) ([]models.RankingItem, error) {
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
			item.Detail = p.Spot.Name + " · " + p.Member.Nickname
			item.Emoji = p.Spot.Emoji
			item.Area = p.Spot.Area
			item.SpotID = &p.Spot.ID
			if item.Name == "" {
				item.Name = p.Spot.Name
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
