package models

import "time"

// GpsVisitResponse is a footprint as one member sees it. Latitude and
// Longitude are null unless the viewer owns the visit or its owner made the
// exact position public; services.GpsVisitService decides that.
type GpsVisitResponse struct {
	ID          string        `json:"id" doc:"足あと ID"`
	Member      MemberSummary `json:"member" doc:"記録したメンバー"`
	IsMine      bool          `json:"is_mine" doc:"自分の足あとなら true"`
	SpotID      *string       `json:"spot_id" doc:"スポット ID"`
	PlaceName   string        `json:"place_name" doc:"場所の名前"`
	Emoji       string        `json:"emoji" doc:"絵文字"`
	Area        string        `json:"area" doc:"地域"`
	VisitedDate string        `json:"visited_date" doc:"訪問日（YYYY-MM-DD）"`
	Weather     *string       `json:"weather" doc:"天気"`
	Temperature *float64      `json:"temperature" doc:"気温（℃）"`
	Note        *string       `json:"note" doc:"感想"`
	IsPublic    bool          `json:"is_public" doc:"全体に公開していれば true"`
	ExactPublic bool          `json:"exact_public" doc:"正確な座標も公開していれば true"`
	Latitude    *float64      `json:"latitude" doc:"緯度（自分の足あと、または exact_public=true のときのみ。それ以外は null）"`
	Longitude   *float64      `json:"longitude" doc:"経度（同上）"`
	CreatedAt   time.Time     `json:"created_at" doc:"記録日時"`
}

type ListGpsVisitsOutput struct {
	Body struct {
		Visits []GpsVisitResponse `json:"visits" doc:"足あと（訪問日の新しい順）"`
	}
}

type CreateGpsVisitInput struct {
	Body struct {
		SpotID      string   `json:"spot_id,omitempty" maxLength:"36" doc:"スポット ID（任意）"`
		PlaceName   string   `json:"place_name,omitempty" maxLength:"100" doc:"場所の名前（spot_id がなければ必須）"`
		Emoji       string   `json:"emoji,omitempty" maxLength:"10" doc:"絵文字（省略時はスポットの絵文字か 📍）"`
		Area        string   `json:"area,omitempty" maxLength:"20" doc:"地域（省略時はスポットの地域か「未設定」）"`
		VisitedDate string   `json:"visited_date" pattern:"^[0-9]{4}-[0-9]{2}-[0-9]{2}$" doc:"訪問日（YYYY-MM-DD）"`
		Weather     string   `json:"weather,omitempty" maxLength:"20" doc:"天気"`
		Temperature *float64 `json:"temperature,omitempty" minimum:"-60" maximum:"60" doc:"気温（℃）"`
		Note        string   `json:"note,omitempty" maxLength:"2000" doc:"感想"`
		IsPublic    *bool    `json:"is_public,omitempty" default:"false" doc:"全体に公開する（省略時 false: 明示しない限り非公開）"`
		ExactPublic *bool    `json:"exact_public,omitempty" doc:"正確な座標も公開する（省略時 false）"`
		Latitude    *float64 `json:"latitude,omitempty" minimum:"-90" maximum:"90" doc:"緯度"`
		Longitude   *float64 `json:"longitude,omitempty" minimum:"-180" maximum:"180" doc:"経度"`
	}
}

type GpsVisitOutput struct {
	Body struct {
		Visit GpsVisitResponse `json:"visit" doc:"足あと"`
	}
}

type GpsVisitIDInput struct {
	ID string `path:"id" maxLength:"36" doc:"足あと ID"`
}

type UpdateGpsVisitVisibilityInput struct {
	ID   string `path:"id" maxLength:"36" doc:"足あと ID"`
	Body struct {
		IsPublic    bool  `json:"is_public" doc:"全体に公開する"`
		ExactPublic *bool `json:"exact_public,omitempty" doc:"正確な座標も公開する（省略時は変更しない）"`
	}
}
