package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestSpotScore(t *testing.T) {
	// visits + likes + saves*3, as in the prototype.
	cases := []struct{ visits, likes, saves, want int }{
		{0, 0, 0, 0},
		{128, 42, 0, 170},
		{2, 3, 4, 17},
		{0, 0, 1, 3},
	}
	for _, c := range cases {
		if got := SpotScore(c.visits, c.likes, c.saves); got != c.want {
			t.Errorf("SpotScore(%d, %d, %d) = %d, want %d", c.visits, c.likes, c.saves, got, c.want)
		}
	}
}

func TestRankingTitle(t *testing.T) {
	cases := map[string]string{
		"朝の苔の森を歩く\n静かな森の空気が気持ちよかった。": "朝の苔の森を歩く",
		"  one line  ":                  "one line",
		"":                              "",
		strings.Repeat("あ", 40) + "\nx": strings.Repeat("あ", 28),
	}
	for caption, want := range cases {
		if got := RankingTitle(caption); got != want {
			t.Errorf("RankingTitle(%q) = %q, want %q", caption, got, want)
		}
	}
}

func TestParseTags(t *testing.T) {
	got, err := ParseTags(" 自然, カフェ、秋旅,,#自然 , ＃湖 ，cafe")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"自然", "カフェ", "秋旅", "湖", "cafe"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %q, want %q", got, want)
	}
	if got, err := ParseTags("  "); err != nil || len(got) != 0 {
		t.Fatalf("blank = %q, %v", got, err)
	}
	var validation *ValidationError
	if _, err := ParseTags(strings.Repeat("あ", 51)); !errors.As(err, &validation) {
		t.Fatalf("long tag: err = %v", err)
	}
	if _, err := ParseTags("a,b,c,d,e,f,g,h,i,j,k"); !errors.As(err, &validation) {
		t.Fatalf("11 tags: err = %v", err)
	}
}

// trackingUploader is a storage.ImageUploader that records what it stored
// and deleted. afterUpload, when set, runs once an upload has succeeded.
type trackingUploader struct {
	mu          sync.Mutex
	stored      map[string]bool
	deleted     []string
	n           int
	afterUpload func()
}

func newTrackingUploader() *trackingUploader {
	return &trackingUploader{stored: map[string]bool{}}
}

func (u *trackingUploader) UploadImage(_ context.Context, kind string, r io.Reader) (string, error) {
	data, _ := io.ReadAll(r)
	if DetectImageType(data) == "" {
		return "", storage.ErrInvalidImage
	}
	u.mu.Lock()
	u.n++
	url := "http://storage.test/" + kind + "/" + string(rune('a'+u.n)) + ".jpg"
	u.stored[url] = true
	u.mu.Unlock()
	if u.afterUpload != nil {
		u.afterUpload()
	}
	return url, nil
}

func (u *trackingUploader) DeleteImage(_ context.Context, url string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.stored, url)
	u.deleted = append(u.deleted, url)
	return nil
}

func addMember(t *testing.T, db *gorm.DB, nickname string) string {
	t.Helper()
	m := models.Member{Nickname: nickname, PreferredLanguage: "ja", TransferCodeHash: HashTransferCode(nickname + "-code"), SessionVersion: 1}
	if err := db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	return m.ID
}

func addSpot(t *testing.T, db *gorm.DB, name, area, category string) string {
	t.Helper()
	s := models.Spot{Name: name, Emoji: "🌲", Area: area, Category: category, Description: name}
	if err := db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s.ID
}

var postClock = time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)

// mustCreatePost creates a post and gives it a created_at one minute after
// the previous one, so "newest first" is the creation order whatever the
// speed of the machine.
func mustCreatePost(t *testing.T, s *PostService, memberID string, p CreatePostParams) *models.PostResponse {
	t.Helper()
	post, err := s.Create(context.Background(), memberID, p)
	if err != nil {
		t.Fatal(err)
	}
	postClock = postClock.Add(time.Minute)
	if err := s.db.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("created_at", postClock).Error; err != nil {
		t.Fatal(err)
	}
	return post
}

func TestToggleLikeAndBookmarkIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewPostService(db, newTrackingUploader())
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	post := mustCreatePost(t, s, alice, CreatePostParams{Caption: "hello"})

	// Like, then unlike: the second toggle undoes the first.
	liked, count, err := s.ToggleLike(ctx, bob, post.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("first toggle = %v, %d, %v", liked, count, err)
	}
	liked, count, err = s.ToggleLike(ctx, bob, post.ID)
	if err != nil || liked || count != 0 {
		t.Fatalf("second toggle = %v, %d, %v", liked, count, err)
	}

	// Two members' likes are counted separately.
	s.ToggleLike(ctx, alice, post.ID)
	if _, count, _ = s.ToggleLike(ctx, bob, post.ID); count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	// The viewer's own state is per member.
	got, _ := s.Get(ctx, bob, post.ID)
	if !got.IsLiked || got.LikeCount != 2 || got.IsBookmarked || got.IsMine {
		t.Fatalf("bob sees %+v", got)
	}

	// A like that already exists cannot be duplicated, even inserted directly.
	if err := db.Create(&models.Like{MemberID: bob, PostID: post.ID}).Error; err == nil {
		t.Fatal("a second like row for the same member and post was accepted")
	}

	bookmarked, err := s.ToggleBookmark(ctx, bob, post.ID)
	if err != nil || !bookmarked {
		t.Fatalf("bookmark = %v, %v", bookmarked, err)
	}
	saved, _ := s.Bookmarked(ctx, bob)
	if len(saved) != 1 || saved[0].ID != post.ID || !saved[0].IsBookmarked {
		t.Fatalf("bob's bookmarks = %+v", saved)
	}
	if others, _ := s.Bookmarked(ctx, alice); len(others) != 0 {
		t.Fatalf("alice sees bob's bookmark: %+v", others)
	}
	bookmarked, _ = s.ToggleBookmark(ctx, bob, post.ID)
	if saved, _ = s.Bookmarked(ctx, bob); bookmarked || len(saved) != 0 {
		t.Fatalf("bookmark not removed: %v, %+v", bookmarked, saved)
	}

	// Posts that do not exist.
	if _, _, err := s.ToggleLike(ctx, bob, "no-such-post"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("like on a missing post: %v", err)
	}
	if _, err := s.ToggleBookmark(ctx, bob, "no-such-post"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("bookmark on a missing post: %v", err)
	}
}

func TestRankingsIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewPostService(db, newTrackingUploader())
	alice, bob, carol := addMember(t, db, "Alice"), addMember(t, db, "Bob"), addMember(t, db, "Carol")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")
	castle := addSpot(t, db, "松本城", "中信", "歴史")
	lake := addSpot(t, db, "諏訪湖", "南信", "自然")
	lodging := addSpot(t, db, "佐久の家", "東信", "宿泊")

	pondPost := mustCreatePost(t, s, alice, CreatePostParams{Caption: "苔の森\n本文", SpotID: pond})
	castlePost := mustCreatePost(t, s, bob, CreatePostParams{Caption: "城下町", SpotID: castle})
	mustCreatePost(t, s, bob, CreatePostParams{Caption: "湖", SpotID: lake})
	lodgingPost := mustCreatePost(t, s, carol, CreatePostParams{Caption: "宿", SpotID: lodging})
	freePost := mustCreatePost(t, s, carol, CreatePostParams{Caption: "スポットなし"})

	// pond: 1 like + 1 save*3 + 1 public visit = 5. The private visit does not count.
	s.ToggleLike(ctx, bob, pondPost.ID)
	s.ToggleBookmark(ctx, carol, pondPost.ID)
	visits := NewGpsVisitService(db)
	yes, no := true, false
	for _, public := range []*bool{&yes, &no} {
		if _, err := visits.Create(ctx, alice, CreateGpsVisitParams{SpotID: pond, VisitedDate: "2026-10-01", IsPublic: public}); err != nil {
			t.Fatal(err)
		}
	}
	// castle: 3 likes = 3.
	for _, m := range []string{alice, bob, carol} {
		s.ToggleLike(ctx, m, castlePost.ID)
	}
	// The lodging would win with 4 points per member, but is not ranked.
	for _, m := range []string{alice, bob, carol} {
		s.ToggleLike(ctx, m, lodgingPost.ID)
		s.ToggleBookmark(ctx, m, lodgingPost.ID)
	}
	s.ToggleLike(ctx, alice, freePost.ID)

	spots, err := s.SpotRanking(ctx, "", 3)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, item := range spots {
		got = append(got, item.Name)
	}
	if want := []string{"白駒の池", "松本城", "諏訪湖"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("spot ranking = %v, want %v", got, want)
	}
	if spots[0].Rank != 1 || spots[0].Score != 5 || spots[0].ID != pond || spots[0].PostCount != 1 || spots[0].LikeCount != 1 || spots[0].Detail != "東信 · 自然" {
		t.Fatalf("first = %+v", spots[0])
	}
	if spots[1].Score != 3 || spots[1].Rank != 2 || spots[2].Score != 0 || spots[2].Rank != 3 {
		t.Fatalf("scores = %d, %d", spots[1].Score, spots[2].Score)
	}
	if two, _ := s.SpotRanking(ctx, "", 2); len(two) != 2 {
		t.Fatalf("limit 2 returned %d", len(two))
	}

	posts, err := s.PostRanking(ctx, "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 || posts[0].Score != 3 || posts[1].Score != 3 || posts[2].Score != 1 {
		t.Fatalf("post ranking = %+v", posts)
	}
	// Equal like counts: the newer post first.
	if posts[0].ID != lodgingPost.ID || posts[1].ID != castlePost.ID {
		t.Fatalf("tie order = %s, %s", posts[0].Name, posts[1].Name)
	}
	if posts[1].Name != "城下町" || posts[1].Detail != "松本城 · Bob" || posts[1].SpotID == nil || *posts[1].SpotID != castle {
		t.Fatalf("post item = %+v", posts[1])
	}
	// A post's title is the first line of its caption.
	if posts[2].ID != pondPost.ID && posts[2].ID != freePost.ID {
		t.Fatalf("third = %+v", posts[2])
	}
}

func TestCreatePostIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	uploader := newTrackingUploader()
	s := NewPostService(db, uploader)
	alice := addMember(t, db, "Alice")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")
	jpeg := testutil.PlainJPEG(t, 8, 8)

	post := mustCreatePost(t, s, alice, CreatePostParams{
		Caption: "  朝の森  ", SpotID: pond, Language: "ja", Weather: "晴れ", Temperature: "13.5",
		VisitedAt: "2026-10-08", Area: "南信", Tags: "自然, 朝さんぽ", Media: bytes.NewReader(jpeg),
	})
	if post.Caption != "朝の森" || post.Spot == nil || post.Spot.Name != "白駒の池" || *post.Weather != "晴れ" || *post.Temperature != 13.5 ||
		*post.VisitedAt != "2026-10-08" || !reflect.DeepEqual(post.Tags, []string{"自然", "朝さんぽ"}) || !post.IsMine || post.IsSample {
		t.Fatalf("post = %+v", post)
	}
	// The spot's area wins over the form's.
	if *post.Area != "東信" {
		t.Fatalf("area = %q, want the spot's", *post.Area)
	}
	if len(post.Media) != 1 || !uploader.stored[post.Media[0].URL] || post.Media[0].Type != "photo" {
		t.Fatalf("media = %+v", post.Media)
	}

	// Photo only, and caption only, are both fine; neither is not.
	if p := mustCreatePost(t, s, alice, CreatePostParams{Media: bytes.NewReader(jpeg), Area: "北信"}); p.Caption != "" || *p.Area != "北信" || p.Spot != nil {
		t.Fatalf("photo-only post = %+v", p)
	}
	var validation *ValidationError
	for name, p := range map[string]CreatePostParams{
		"nothing":      {},
		"blank":        {Caption: "   "},
		"long caption": {Caption: strings.Repeat("あ", 2001)},
		"language":     {Caption: "x", Language: "fr"},
		"temperature":  {Caption: "x", Temperature: "warm"},
		"hot":          {Caption: "x", Temperature: "999"},
		"date":         {Caption: "x", VisitedAt: "10/08/2026"},
		"area":         {Caption: "x", Area: "関東"},
		"spot":         {Caption: "x", SpotID: "no-such-spot"},
		"weather":      {Caption: "x", Weather: strings.Repeat("晴", 21)},
	} {
		if _, err := s.Create(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("%s: err = %v, want a ValidationError", name, err)
		}
	}

	// A file that is not an image: no post, nothing stored.
	var before, after int64
	db.Model(&models.Post{}).Count(&before)
	storedBefore := len(uploader.stored)
	_, err := s.Create(ctx, alice, CreatePostParams{Caption: "x", Media: strings.NewReader("#!/bin/sh\nrm -rf /")})
	if !errors.Is(err, storage.ErrInvalidImage) {
		t.Fatalf("err = %v, want ErrInvalidImage", err)
	}
	db.Model(&models.Post{}).Count(&after)
	if after != before || len(uploader.stored) != storedBefore {
		t.Fatal("a rejected upload left a post or an object behind")
	}
}

// If the insert fails after the photo was stored, the post is not created
// and the stored object is removed again.
func TestCreatePostFailureRemovesImageIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	uploader := newTrackingUploader()
	s := NewPostService(db, uploader)
	alice := addMember(t, db, "Alice")

	// Cancel the request right after the upload: the transaction then fails.
	ctx, cancel := context.WithCancel(context.Background())
	uploader.afterUpload = cancel

	_, err := s.Create(ctx, alice, CreatePostParams{Caption: "x", Tags: "a,b", Media: bytes.NewReader(testutil.PlainJPEG(t, 8, 8))})
	if err == nil {
		t.Fatal("expected the insert to fail")
	}
	if uploader.n != 1 {
		t.Fatalf("uploads = %d, want 1", uploader.n)
	}
	if len(uploader.stored) != 0 || len(uploader.deleted) != 1 {
		t.Fatalf("the uploaded object was left behind: stored %v, deleted %v", uploader.stored, uploader.deleted)
	}
	for _, model := range []any{&models.Post{}, &models.PostMedia{}, &models.PostTag{}} {
		var n int64
		db.Model(model).Count(&n)
		if n != 0 {
			t.Fatalf("%T has %d rows after a failed create", model, n)
		}
	}
}

func TestDeletePostIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	uploader := newTrackingUploader()
	s := NewPostService(db, uploader)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")

	post := mustCreatePost(t, s, alice, CreatePostParams{Caption: "x", Tags: "a,b", Media: bytes.NewReader(testutil.PlainJPEG(t, 8, 8))})
	other := mustCreatePost(t, s, alice, CreatePostParams{Caption: "stays"})
	s.ToggleLike(ctx, bob, post.ID)
	s.ToggleBookmark(ctx, bob, post.ID)
	s.ToggleBookmark(ctx, bob, other.ID)
	url := post.Media[0].URL

	// Not the owner: nothing changes.
	if err := s.Delete(ctx, bob, post.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bob deleting alice's post: %v", err)
	}
	if _, err := s.Get(ctx, alice, post.ID); err != nil || !uploader.stored[url] {
		t.Fatalf("post or image gone after a refused delete: %v", err)
	}
	if err := s.Delete(ctx, alice, "no-such-post"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("missing post: %v", err)
	}

	if err := s.Delete(ctx, alice, post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, alice, post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("get after delete: %v", err)
	}
	// Dependent rows went with it, and the stored image was deleted.
	for _, model := range []any{&models.PostMedia{}, &models.PostTag{}, &models.Like{}} {
		var n int64
		db.Model(model).Where("post_id = ?", post.ID).Count(&n)
		if n != 0 {
			t.Fatalf("%T still has %d rows of the deleted post", model, n)
		}
	}
	if uploader.stored[url] || len(uploader.deleted) != 1 || uploader.deleted[0] != url {
		t.Fatalf("stored image not deleted: %v", uploader.deleted)
	}

	// Listings that referred to it keep working.
	saved, err := s.Bookmarked(ctx, bob)
	if err != nil || len(saved) != 1 || saved[0].ID != other.ID {
		t.Fatalf("bob's bookmarks = %+v, %v", saved, err)
	}
	if feed, total, err := s.List(ctx, bob, PostFilter{}); err != nil || total != 1 || len(feed) != 1 {
		t.Fatalf("feed = %d of %d, %v", len(feed), total, err)
	}
	if _, err := s.PostRanking(ctx, "", 3); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, alice, post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestListPostsFiltersIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewPostService(db, newTrackingUploader())
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "旅するKen")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")

	a := mustCreatePost(t, s, alice, CreatePostParams{Caption: "苔の森", SpotID: pond, Weather: "晴れ", VisitedAt: "2026-10-08", Tags: "自然"})
	b := mustCreatePost(t, s, bob, CreatePostParams{Caption: "100%_great café", Area: "中信", Weather: "くもり", VisitedAt: "2026-09-29", Tags: "カフェ"})
	c := mustCreatePost(t, s, bob, CreatePostParams{Caption: "湖畔", Area: "南信", Weather: "晴れ", VisitedAt: "2026-09-22"})
	s.ToggleLike(ctx, alice, c.ID)

	ids := func(f PostFilter) []string {
		t.Helper()
		posts, total, err := s.List(ctx, alice, f)
		if err != nil {
			t.Fatal(err)
		}
		if f.Limit == 0 && int(total) != len(posts) {
			t.Fatalf("total = %d, posts = %d", total, len(posts))
		}
		out := []string{}
		for _, p := range posts {
			out = append(out, p.ID)
		}
		return out
	}
	check := func(name string, f PostFilter, want ...string) {
		t.Helper()
		if want == nil {
			want = []string{}
		}
		if got := ids(f); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}

	check("latest", PostFilter{}, c.ID, b.ID, a.ID)
	check("popular", PostFilter{Sort: "popular"}, c.ID, b.ID, a.ID)
	check("area from spot", PostFilter{Area: "東信"}, a.ID)
	check("area from form", PostFilter{Area: "中信"}, b.ID)
	check("weather", PostFilter{Weather: "晴れ"}, c.ID, a.ID)
	check("month", PostFilter{Month: "2026-09"}, c.ID, b.ID)
	check("month and weather", PostFilter{Month: "2026-09", Weather: "晴れ"}, c.ID)
	check("search caption", PostFilter{Query: "苔"}, a.ID)
	check("search spot", PostFilter{Query: "白駒"}, a.ID)
	check("search tag", PostFilter{Query: "カフェ"}, b.ID)
	check("search nickname", PostFilter{Query: "ken"}, c.ID, b.ID)
	// LIKE wildcards in the search text are literal.
	check("search percent", PostFilter{Query: "100%_g"}, b.ID)
	check("search underscore only", PostFilter{Query: "_"}, b.ID)
	check("search wildcard only", PostFilter{Query: "%"}, b.ID)
	check("no match", PostFilter{Query: "存在しない"})

	posts, total, err := s.List(ctx, alice, PostFilter{Limit: 2, Offset: 1})
	if err != nil || total != 3 || len(posts) != 2 || posts[0].ID != b.ID {
		t.Fatalf("paging = %d of %d, %v", len(posts), total, err)
	}

	mine, _ := s.Mine(ctx, bob)
	if len(mine) != 2 || mine[0].ID != c.ID || !mine[0].IsMine {
		t.Fatalf("bob's posts = %+v", mine)
	}
}

// How "inserted" is told from "was already there": GORM's OnConflict
// DoNothing becomes INSERT ... ON DUPLICATE KEY UPDATE id = id on MySQL, and
// the server reports 1 affected row for an insert and 0 for a duplicate left
// unchanged. toggle depends on exactly that.
func TestToggleInsertDetectionIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	s := NewPostService(db, newTrackingUploader())
	alice := addMember(t, db, "Alice")
	post := mustCreatePost(t, s, alice, CreatePostParams{Caption: "x"})

	insert := func() int64 {
		t.Helper()
		res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.Like{MemberID: alice, PostID: post.ID})
		if res.Error != nil {
			t.Fatal(res.Error)
		}
		return res.RowsAffected
	}
	if n := insert(); n != 1 {
		t.Fatalf("first insert: RowsAffected = %d, want 1", n)
	}
	// A fresh struct with a new id, same (member_id, post_id): the duplicate.
	if n := insert(); n != 0 {
		t.Fatalf("duplicate insert: RowsAffected = %d, want 0", n)
	}
	var rows int64
	db.Model(&models.Like{}).Count(&rows)
	if rows != 1 {
		t.Fatalf("rows = %d, want 1", rows)
	}

	// And through the service: on, off, on.
	db.Where("1 = 1").Delete(&models.Like{})
	for i, want := range []bool{true, false, true} {
		liked, count, err := s.ToggleLike(context.Background(), alice, post.ID)
		wantCount := 0
		if want {
			wantCount = 1
		}
		if err != nil || liked != want || count != wantCount {
			t.Fatalf("toggle %d = %v, %d, %v; want %v, %d", i+1, liked, count, err, want, wantCount)
		}
	}
}

// Many members reacting to one post at the same moment, and one member
// hammering the button, must never fail: no deadlock reaches the caller,
// and what the calls report matches what is in the table.
func TestToggleConcurrencyIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewPostService(db, newTrackingUploader())
	author := addMember(t, db, "Author")
	post := mustCreatePost(t, s, author, CreatePostParams{Caption: "popular"})

	const members = 24
	ids := make([]string, members)
	for i := range ids {
		ids[i] = addMember(t, db, fmt.Sprintf("Member%02d", i))
	}

	type result struct {
		member string
		on     bool
		err    error
	}
	run := func(n int, call func(i int) result) []result {
		t.Helper()
		results := make([]result, n)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				results[i] = call(i)
			}()
		}
		close(start)
		wg.Wait()
		for _, r := range results {
			if r.err != nil {
				t.Fatalf("a concurrent toggle failed: %v", r.err)
			}
		}
		return results
	}
	count := func(model any, where string, args ...any) int {
		t.Helper()
		var n int64
		if err := db.Model(model).Where(where, args...).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		return int(n)
	}

	// Round 1: every member likes and bookmarks the post at once. This is
	// the case that deadlocked when the toggle deleted before inserting.
	for round := 1; round <= 3; round++ {
		wantOn := round%2 == 1
		likes := run(members, func(i int) result {
			on, _, err := s.ToggleLike(ctx, ids[i], post.ID)
			return result{ids[i], on, err}
		})
		saves := run(members, func(i int) result {
			on, err := s.ToggleBookmark(ctx, ids[i], post.ID)
			return result{ids[i], on, err}
		})
		for _, r := range append(likes, saves...) {
			if r.on != wantOn {
				t.Fatalf("round %d: a toggle reported %v, want %v", round, r.on, wantOn)
			}
		}
		want := 0
		if wantOn {
			want = members
		}
		if got := count(&models.Like{}, "post_id = ?", post.ID); got != want {
			t.Fatalf("round %d: %d like rows, want %d", round, got, want)
		}
		if got := count(&models.Bookmark{}, "post_id = ?", post.ID); got != want {
			t.Fatalf("round %d: %d bookmark rows, want %d", round, got, want)
		}
	}
	// After three rounds everyone has it on.

	// Round 2: one member toggles 40 times in parallel while the others
	// toggle once each. The member can never hold two rows, each of their
	// calls flipped the state exactly once (as many "on" as "off" for an
	// even number of calls, ending where it started), and the count the
	// service reports afterwards is the number of rows.
	const presses = 40
	hammer := ids[0]
	before := count(&models.Like{}, "member_id = ? AND post_id = ?", hammer, post.ID)
	results := run(presses+members-1, func(i int) result {
		if i < presses {
			on, _, err := s.ToggleLike(ctx, hammer, post.ID)
			return result{hammer, on, err}
		}
		m := ids[i-presses+1]
		on, _, err := s.ToggleLike(ctx, m, post.ID)
		return result{m, on, err}
	})
	ons := 0
	for _, r := range results[:presses] {
		if r.on {
			ons++
		}
	}
	if ons != presses/2 {
		t.Fatalf("%d presses reported on %d times, want %d: some toggle did not flip the state", presses, ons, presses/2)
	}
	after := count(&models.Like{}, "member_id = ? AND post_id = ?", hammer, post.ID)
	if after != before || after > 1 {
		t.Fatalf("the member holds %d likes after an even number of presses, had %d", after, before)
	}
	// The others each toggled once more: off.
	rows := count(&models.Like{}, "post_id = ?", post.ID)
	if rows != after {
		t.Fatalf("%d like rows, want %d (only the hammering member's)", rows, after)
	}
	var duplicates int64
	db.Raw("SELECT COUNT(*) FROM (SELECT member_id FROM likes WHERE post_id = ? GROUP BY member_id HAVING COUNT(*) > 1) d", post.ID).Scan(&duplicates)
	if duplicates != 0 {
		t.Fatalf("%d members hold more than one like", duplicates)
	}
	// The count a caller is told is the count in the table.
	_, reported, err := s.ToggleLike(ctx, author, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(&models.Like{}, "post_id = ?", post.ID); reported != got || got != rows+1 {
		t.Fatalf("reported like_count %d, rows %d, want %d", reported, got, rows+1)
	}
}

func TestRetryOnLockConflict(t *testing.T) {
	deadlock := &mysqldriver.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock"}
	timeout := &mysqldriver.MySQLError{Number: 1205, Message: "Lock wait timeout exceeded"}
	duplicate := &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry"}
	ctx := context.Background()

	// Conflicts are retried until the transaction goes through.
	calls := 0
	err := retryOnLockConflict(ctx, func() error {
		calls++
		switch calls {
		case 1:
			return fmt.Errorf("toggle: %w", deadlock)
		case 2:
			return timeout
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("err = %v after %d calls, want nil after 3", err, calls)
	}

	// Bounded: a conflict that never clears is returned after the last try.
	calls = 0
	err = retryOnLockConflict(ctx, func() error { calls++; return deadlock })
	if !errors.Is(err, deadlock) || calls != lockRetryAttempts {
		t.Fatalf("err = %v after %d calls, want the deadlock after %d", err, calls, lockRetryAttempts)
	}

	// Anything else is not retried.
	for _, other := range []error{duplicate, ErrPostNotFound, errors.New("boom")} {
		calls = 0
		if err := retryOnLockConflict(ctx, func() error { calls++; return other }); !errors.Is(err, other) || calls != 1 {
			t.Fatalf("%v: %d calls, err %v; want 1 call", other, calls, err)
		}
	}

	// A cancelled request stops waiting.
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	calls = 0
	if err := retryOnLockConflict(cancelled, func() error { calls++; return deadlock }); !errors.Is(err, deadlock) || calls != 1 {
		t.Fatalf("cancelled: %d calls, err %v", calls, err)
	}
}
