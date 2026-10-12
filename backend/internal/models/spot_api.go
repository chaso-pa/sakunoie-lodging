package models

import "time"

type SpotResponse struct {
	ID          string `json:"id" doc:"スポット ID"`
	Name        string `json:"name" doc:"スポット名"`
	Emoji       string `json:"emoji" doc:"絵文字"`
	Area        string `json:"area" doc:"地域（東信・北信・中信・南信）"`
	Category    string `json:"category" doc:"カテゴリ"`
	Description string `json:"description" doc:"説明"`
	IsSample    bool   `json:"is_sample" doc:"サンプルデータなら true"`
}

func NewSpotResponse(s *Spot) SpotResponse {
	return SpotResponse{
		ID:          s.ID,
		Name:        s.Name,
		Emoji:       s.Emoji,
		Area:        s.Area,
		Category:    s.Category,
		Description: s.Description,
		IsSample:    s.IsSample,
	}
}

type SpotListItem struct {
	SpotResponse
	ReviewCount int     `json:"review_count" doc:"口コミ数"`
	AvgStars    float64 `json:"avg_stars" doc:"口コミの平均（口コミがなければ 0）"`
	PostCount   int     `json:"post_count" doc:"関連投稿数"`
}

type ListSpotsInput struct {
	Area string `query:"area" maxLength:"20" doc:"地域で絞り込む（任意）"`
}

type ListSpotsOutput struct {
	Body struct {
		Spots []SpotListItem `json:"spots" doc:"スポット一覧"`
	}
}

type ReviewResponse struct {
	ID        string        `json:"id" doc:"口コミ ID"`
	Member    MemberSummary `json:"member" doc:"投稿者"`
	Stars     int           `json:"stars" doc:"星（1〜5）"`
	Text      string        `json:"text" doc:"本文"`
	IsSample  bool          `json:"is_sample" doc:"サンプルデータなら true"`
	CreatedAt time.Time     `json:"created_at" doc:"投稿日時"`
}

func NewReviewResponse(r *Review) ReviewResponse {
	return ReviewResponse{
		ID:        r.ID,
		Member:    NewMemberSummary(&r.Member),
		Stars:     r.Stars,
		Text:      r.Text,
		IsSample:  r.IsSample,
		CreatedAt: r.CreatedAt,
	}
}

// SpotPostResponse is a post as listed on a spot's page.
type SpotPostResponse struct {
	ID        string          `json:"id" doc:"投稿 ID"`
	Caption   string          `json:"caption" doc:"キャプション"`
	Media     []MediaResponse `json:"media" doc:"写真"`
	Member    MemberSummary   `json:"member" doc:"投稿者"`
	LikeCount int             `json:"like_count" doc:"いいね数"`
	IsSample  bool            `json:"is_sample" doc:"サンプルデータなら true"`
	CreatedAt time.Time       `json:"created_at" doc:"投稿日時"`
}

type GetSpotInput struct {
	ID string `path:"id" maxLength:"36" doc:"スポット ID"`
}

type GetSpotOutput struct {
	Body struct {
		Spot    SpotResponse       `json:"spot" doc:"スポット"`
		Reviews []ReviewResponse   `json:"reviews" doc:"口コミ（新しい順）"`
		Posts   []SpotPostResponse `json:"posts" doc:"関連投稿（新しい順）"`
	}
}

type CreateReviewInput struct {
	ID   string `path:"id" maxLength:"36" doc:"スポット ID"`
	Body struct {
		Stars int    `json:"stars" minimum:"1" maximum:"5" doc:"星（1〜5）"`
		Text  string `json:"text" minLength:"1" maxLength:"1000" doc:"本文"`
	}
}

type CreateReviewOutput struct {
	Body struct {
		Review ReviewResponse `json:"review" doc:"作成された口コミ"`
	}
}

type AiSpotResponse struct {
	ID          string `json:"id" doc:"AI スポット ID"`
	Name        string `json:"name" doc:"スポット名"`
	Area        string `json:"area" doc:"エリア"`
	Kind        string `json:"kind" doc:"種別"`
	DriveMin    int    `json:"drive_min" doc:"車での所要時間（分）"`
	TransitMin  int    `json:"transit_min" doc:"公共交通での所要時間（分）"`
	StayMin     int    `json:"stay_min" doc:"滞在目安（分）"`
	Description string `json:"description" doc:"説明"`
	MapQuery    string `json:"map_query" doc:"地図検索用の文字列"`
	SearchURL   string `json:"search_url" doc:"詳細検索の URL"`
}

type ListAiSpotsInput struct {
	Genre     string `query:"genre" maxLength:"30" doc:"ジャンル（任意。穴場スポット・自然・絶景・歴史・文化・カフェ・グルメ）"`
	Purpose   string `query:"purpose" default:"all" enum:"all,karuizawa,onsen,hiking,japan,scenic,rainyday,family" doc:"目的"`
	Weather   string `query:"weather" default:"any" enum:"any,sunny,rainy" doc:"天気"`
	Duration  string `query:"duration" default:"half" enum:"short,half,full" doc:"所要時間（件数: short=2, half=3, full=5）"`
	Transport string `query:"transport" default:"car" enum:"car,transit" doc:"移動手段（並び順に使う）"`
}

type ListAiSpotsOutput struct {
	Body struct {
		Spots []AiSpotResponse `json:"spots" doc:"おすすめスポット"`
	}
}
