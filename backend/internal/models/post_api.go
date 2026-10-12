package models

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type PostSpotResponse struct {
	ID    string `json:"id" doc:"スポット ID"`
	Name  string `json:"name" doc:"スポット名"`
	Emoji string `json:"emoji" doc:"絵文字"`
	Area  string `json:"area" doc:"地域"`
}

// PostResponse is a post as one member sees it: is_liked, is_bookmarked and
// is_mine depend on who is asking.
type PostResponse struct {
	ID           string            `json:"id" doc:"投稿 ID"`
	Member       MemberSummary     `json:"member" doc:"投稿者"`
	Spot         *PostSpotResponse `json:"spot" doc:"スポット（未指定なら null）"`
	Caption      string            `json:"caption" doc:"キャプション"`
	Media        []MediaResponse   `json:"media" doc:"写真"`
	Language     string            `json:"language" doc:"投稿言語"`
	Area         *string           `json:"area" doc:"地域（スポットの地域、なければ投稿時に選んだ地域）"`
	Weather      *string           `json:"weather" doc:"天気"`
	Temperature  *float64          `json:"temperature" doc:"気温（℃）"`
	VisitedAt    *string           `json:"visited_at" doc:"訪問日（YYYY-MM-DD）"`
	Tags         []string          `json:"tags" doc:"タグ"`
	LikeCount    int               `json:"like_count" doc:"いいね数"`
	IsLiked      bool              `json:"is_liked" doc:"自分がいいね済みなら true"`
	IsBookmarked bool              `json:"is_bookmarked" doc:"自分が保存済みなら true"`
	IsMine       bool              `json:"is_mine" doc:"自分の投稿なら true"`
	IsSample     bool              `json:"is_sample" doc:"サンプルデータなら true"`
	CreatedAt    time.Time         `json:"created_at" doc:"投稿日時"`
}

type ListPostsInput struct {
	Area    string `query:"area" maxLength:"20" doc:"地域で絞り込む"`
	Weather string `query:"weather" maxLength:"20" doc:"天気で絞り込む（完全一致）"`
	Month   string `query:"month" pattern:"^$|^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"訪問月で絞り込む（YYYY-MM）"`
	Query   string `query:"q" maxLength:"100" doc:"キャプション・スポット名・タグ・ニックネーム・天気を検索"`
	Sort    string `query:"sort" default:"latest" enum:"latest,popular" doc:"並び順"`
	Limit   int    `query:"limit" default:"20" minimum:"1" maximum:"50" doc:"件数"`
	Offset  int    `query:"offset" default:"0" minimum:"0" doc:"開始位置"`
}

type ListPostsOutput struct {
	Body struct {
		Posts []PostResponse `json:"posts" doc:"投稿一覧"`
		Total int            `json:"total" doc:"条件に合う投稿の総数"`
	}
}

// PostListOutput is the body of the bookmarked and own post lists.
type PostListOutput struct {
	Body struct {
		Posts []PostResponse `json:"posts" doc:"投稿一覧（新しい順）"`
	}
}

type PostIDInput struct {
	ID string `path:"id" maxLength:"36" doc:"投稿 ID"`
}

type PostOutput struct {
	Body struct {
		Post PostResponse `json:"post" doc:"投稿"`
	}
}

type CreatePostFormData struct {
	Caption     string        `form:"caption" maxLength:"2000" doc:"キャプション（写真がなければ必須）" required:"false"`
	SpotID      string        `form:"spot_id" maxLength:"36" doc:"スポットID（任意）" required:"false"`
	Media       huma.FormFile `form:"media" contentType:"image/jpeg,image/png,image/webp" doc:"写真（任意、1枚）" required:"false"`
	Language    string        `form:"language" default:"ja" enum:"ja,en,ko,zh" doc:"投稿言語" required:"false"`
	Weather     string        `form:"weather" maxLength:"20" doc:"天気" required:"false"`
	Temperature string        `form:"temperature" maxLength:"10" doc:"気温（数値文字列）" required:"false"`
	VisitedAt   string        `form:"visited_at" maxLength:"10" doc:"訪問日（YYYY-MM-DD）" required:"false"`
	Area        string        `form:"area" maxLength:"20" doc:"地域（スポット未指定のとき使う）" required:"false"`
	Tags        string        `form:"tags" maxLength:"300" doc:"タグ（カンマ区切り）" required:"false"`
}

type CreatePostInput struct {
	RawBody huma.MultipartFormFiles[CreatePostFormData]
}

type ToggleLikeOutput struct {
	Body struct {
		Liked     bool `json:"liked" doc:"操作後にいいね済みなら true"`
		LikeCount int  `json:"like_count" doc:"操作後のいいね数"`
	}
}

type ToggleBookmarkOutput struct {
	Body struct {
		Bookmarked bool `json:"bookmarked" doc:"操作後に保存済みなら true"`
	}
}

type RankingItem struct {
	Rank      int     `json:"rank" doc:"順位（1 始まり）"`
	ID        string  `json:"id" doc:"スポット ID または投稿 ID"`
	Name      string  `json:"name" doc:"スポット名、または投稿の見出し（キャプションの 1 行目）"`
	Score     int     `json:"score" doc:"スコア（スポット: 公開の足あと数 + いいね数 + 保存数×3、投稿: いいね数）"`
	Detail    string  `json:"detail" doc:"補足（スポット: 地域 · カテゴリ、投稿: スポット名 · 投稿者）"`
	Emoji     string  `json:"emoji" doc:"絵文字（スポット、または投稿のスポット。なければ空文字）"`
	Area      string  `json:"area" doc:"地域（なければ空文字）"`
	PostCount int     `json:"post_count" doc:"関連投稿数（スポットのみ。投稿では 0）"`
	LikeCount int     `json:"like_count" doc:"いいね数"`
	SpotID    *string `json:"spot_id" doc:"スポット ID（投稿にスポットがなければ null）"`
	IsSample  bool    `json:"is_sample" doc:"サンプルデータなら true"`
}

type RankingsInput struct {
	Type  string `query:"type" default:"spots" enum:"spots,posts" doc:"ランキングの種類"`
	Limit int    `query:"limit" default:"3" minimum:"1" maximum:"20" doc:"件数"`
}

type RankingsOutput struct {
	Body struct {
		Items []RankingItem `json:"items" doc:"ランキング"`
	}
}
