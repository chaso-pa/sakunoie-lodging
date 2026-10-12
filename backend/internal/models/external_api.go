package models

type TranslateInput struct {
	Body struct {
		Text       string `json:"text" minLength:"1" maxLength:"2000" doc:"翻訳するテキスト"`
		SourceLang string `json:"source_lang" enum:"ja,en,ko,zh,auto" doc:"元の言語（auto はサーバーが文字種から判定）"`
		TargetLang string `json:"target_lang" enum:"ja,en,ko,zh" doc:"翻訳先の言語"`
	}
}

type TranslateOutput struct {
	Body struct {
		TranslatedText string `json:"translated_text" doc:"翻訳結果（元の言語と翻訳先が同じなら原文）"`
		Provider       string `json:"provider" enum:"mymemory,openrouter,none" doc:"翻訳したプロバイダ（none は翻訳不要だった場合）"`
		Cached         bool   `json:"cached" doc:"キャッシュから返したなら true"`
		SourceLang     string `json:"source_lang" enum:"ja,en,ko,zh" doc:"翻訳元として扱った言語"`
	}
}

type AIPlanInput struct {
	Body struct {
		Message   string `json:"message" minLength:"1" maxLength:"3000" doc:"相談内容"`
		Language  string `json:"language" enum:"ja,en,ko,zh" doc:"回答の言語"`
		Purpose   string `json:"purpose,omitempty" enum:"all,karuizawa,onsen,hiking,japan,scenic,rainyday,family" doc:"目的（省略時 all）"`
		Genre     string `json:"genre,omitempty" maxLength:"30" doc:"ジャンル"`
		Weather   string `json:"weather,omitempty" enum:"any,sunny,rainy" doc:"天気（省略時 any）"`
		Duration  string `json:"duration,omitempty" enum:"short,half,full" doc:"所要時間（省略時 half）"`
		Transport string `json:"transport,omitempty" enum:"car,transit" doc:"移動手段（省略時 car）"`
	}
}

type AIPlanOutput struct {
	Body struct {
		Answer string `json:"answer" doc:"AI の回答（プレーンテキスト）"`
		Model  string `json:"model" doc:"回答したモデル"`
	}
}

type WeatherInput struct {
	Latitude  float64 `query:"latitude" default:"36.2488" minimum:"-90" maximum:"90" doc:"緯度（省略時は佐久市）"`
	Longitude float64 `query:"longitude" default:"138.4769" minimum:"-180" maximum:"180" doc:"経度（省略時は佐久市）"`
}

type WeatherOutput struct {
	Body struct {
		Weather     string  `json:"weather" doc:"天気（晴れ・晴れ時々くもり・くもり・霧・小雨・雨・雪・雷雨）"`
		Temperature float64 `json:"temperature" doc:"気温（℃、整数に丸めた値）"`
		Icon        string  `json:"icon" doc:"絵文字"`
		Code        int     `json:"code" doc:"WMO 天気コード"`
		Source      string  `json:"source" doc:"取得元"`
	}
}
