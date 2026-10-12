package models

// MemberSummary is how a member appears next to their content. It carries
// nothing but the public profile.
type MemberSummary struct {
	ID        string `json:"id" doc:"メンバー ID"`
	Nickname  string `json:"nickname" doc:"ニックネーム"`
	AvatarURL string `json:"avatar_url" doc:"プロフィール画像の URL（未設定なら空文字）"`
}

// NewMemberSummary maps a Member row to its public summary.
func NewMemberSummary(m *Member) MemberSummary {
	s := MemberSummary{ID: m.ID, Nickname: m.Nickname}
	if m.AvatarURL != nil {
		s.AvatarURL = *m.AvatarURL
	}
	return s
}

type MediaResponse struct {
	URL  string `json:"url" doc:"画像の URL"`
	Type string `json:"type" enum:"photo" doc:"メディア種別"`
}

// OKOutput is the body of endpoints that only confirm success.
type OKOutput struct {
	Body struct {
		OK bool `json:"ok" doc:"常に true"`
	}
}

func NewOKOutput() *OKOutput {
	out := &OKOutput{}
	out.Body.OK = true
	return out
}

// DateLayout is the format of date-only fields (visited_at, visited_date).
const DateLayout = "2006-01-02"
