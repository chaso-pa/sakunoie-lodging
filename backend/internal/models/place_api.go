package models

// PlaceResult is one place found by search, nearby search or reverse
// lookup. It is not stored: POST /spots turns a chosen one into a spot.
type PlaceResult struct {
	Source       string   `json:"source" enum:"seed,osm,google,manual" doc:"取得元。seed / manual は登録済みのスポット"`
	ExternalID   *string  `json:"external_id" doc:"取得元での ID（POST /spots に渡す。登録済みの seed / manual は null）"`
	SpotID       *string  `json:"spot_id" doc:"すでにスポットとして登録されていればその ID（そのまま POST /posts の spot_id に使える）"`
	Name         string   `json:"name" doc:"名前（指定した言語。なければ現地語）"`
	OriginalName *string  `json:"original_name" doc:"現地語の名前（name と違うときだけ）"`
	Address      string   `json:"address" doc:"住所または地域名（不明なら空文字）"`
	Category     string   `json:"category" doc:"カテゴリ（日本語のラベル）"`
	CategoryKey  string   `json:"category_key" enum:"food,cafe,shop,sight,history,nature,onsen,lodging,transport,other" doc:"カテゴリのキー"`
	Emoji        string   `json:"emoji" doc:"カテゴリの絵文字"`
	Latitude     *float64 `json:"latitude" doc:"緯度（不明なら null）"`
	Longitude    *float64 `json:"longitude" doc:"経度（不明なら null）"`
	MapURL       *string  `json:"map_url" doc:"地図サービスの URL"`
	DistanceM    *int     `json:"distance_m" doc:"指定した位置からの距離（メートル。周辺検索のみ）"`
	PhotoURL     *string  `json:"photo_url" doc:"写真の URL（現在は常に null。docs/spot-search.md 参照）"`
}

// PlaceAttribution is the credit that must be shown next to results.
type PlaceAttribution struct {
	Text string `json:"text" doc:"表示する帰属表記"`
	URL  string `json:"url" doc:"リンク先"`
}

type SearchPlacesInput struct {
	Body struct {
		Query     string   `json:"query" minLength:"1" maxLength:"100" doc:"検索語（店名・観光地・住所・郵便番号）"`
		Language  string   `json:"language,omitempty" enum:"ja,en,ko,zh" doc:"結果の言語（省略時は会員の表示言語）"`
		Latitude  *float64 `json:"latitude,omitempty" minimum:"-90" maximum:"90" doc:"この位置の近くを優先する（任意。longitude と一緒に指定）"`
		Longitude *float64 `json:"longitude,omitempty" minimum:"-180" maximum:"180" doc:"同上"`
		Limit     int      `json:"limit,omitempty" minimum:"1" maximum:"10" default:"8" doc:"件数"`
	}
}

type NearbyPlacesInput struct {
	Body struct {
		Latitude  float64 `json:"latitude" minimum:"-90" maximum:"90" doc:"緯度"`
		Longitude float64 `json:"longitude" minimum:"-180" maximum:"180" doc:"経度"`
		RadiusM   int     `json:"radius_m,omitempty" minimum:"100" maximum:"2000" default:"800" doc:"半径（メートル）"`
		Category  string  `json:"category,omitempty" enum:"all,food,cafe,shop,sight,nature,onsen,lodging,transport" default:"all" doc:"カテゴリ"`
		Language  string  `json:"language,omitempty" enum:"ja,en,ko,zh" doc:"結果の言語（省略時は会員の表示言語）"`
		Limit     int     `json:"limit,omitempty" minimum:"1" maximum:"20" default:"15" doc:"件数"`
	}
}

type PlaceListOutput struct {
	Body struct {
		Results     []PlaceResult    `json:"results" doc:"見つかった場所"`
		Provider    string           `json:"provider" enum:"osm,google" doc:"検索に使ったサービス"`
		Attribution PlaceAttribution `json:"attribution" doc:"結果のそばに表示する帰属表記"`
	}
}

type ReversePlaceInput struct {
	Body struct {
		Latitude  float64 `json:"latitude" minimum:"-90" maximum:"90" doc:"地図で選んだ位置の緯度"`
		Longitude float64 `json:"longitude" minimum:"-180" maximum:"180" doc:"地図で選んだ位置の経度"`
		Language  string  `json:"language,omitempty" enum:"ja,en,ko,zh" doc:"結果の言語（省略時は会員の表示言語）"`
	}
}

type ReversePlaceOutput struct {
	Body struct {
		Result      *PlaceResult     `json:"result" doc:"その位置の場所（何も分からなければ null）"`
		Provider    string           `json:"provider" enum:"osm,google" doc:"検索に使ったサービス"`
		Attribution PlaceAttribution `json:"attribution" doc:"結果のそばに表示する帰属表記"`
	}
}

type RegisterSpotInput struct {
	Body struct {
		Source     string   `json:"source" enum:"osm,google,manual" doc:"osm / google: 検索結果を登録、manual: 手入力の場所を登録"`
		ExternalID string   `json:"external_id,omitempty" maxLength:"512" doc:"検索結果の external_id（osm / google のとき必須）"`
		Language   string   `json:"language,omitempty" enum:"ja,en,ko,zh" doc:"名前の言語（省略時は会員の表示言語）"`
		Name       string   `json:"name,omitempty" maxLength:"100" doc:"名前（manual のとき必須）"`
		Address    string   `json:"address,omitempty" maxLength:"200" doc:"住所（manual、任意）"`
		Latitude   *float64 `json:"latitude,omitempty" minimum:"-90" maximum:"90" doc:"緯度（manual、任意。longitude と一緒に指定）"`
		Longitude  *float64 `json:"longitude,omitempty" minimum:"-180" maximum:"180" doc:"経度（manual、任意）"`
		Category   string   `json:"category,omitempty" enum:"food,cafe,shop,sight,history,nature,onsen,lodging,transport,other" doc:"カテゴリ（manual、省略時 other）"`
		Area       string   `json:"area,omitempty" enum:"東信,北信,中信,南信" doc:"地域（manual、任意）"`
	}
}

type RegisterSpotOutput struct {
	Body struct {
		Spot    SpotResponse `json:"spot" doc:"登録された、またはすでにあったスポット"`
		Created bool         `json:"created" doc:"新しく行を作ったなら true。同じ場所が登録済みだったなら false"`
	}
}
