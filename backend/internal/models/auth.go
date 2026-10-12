package models

import "github.com/danielgtaylor/huma/v2"

// MemberResponse is the public representation of a member.
type MemberResponse struct {
	ID                string `json:"id" doc:"メンバー ID"`
	Nickname          string `json:"nickname" doc:"ニックネーム"`
	AvatarURL         string `json:"avatar_url" doc:"プロフィール画像の URL（未設定なら空文字）"`
	PreferredLanguage string `json:"preferred_language" enum:"ja,en,ko,zh" doc:"表示言語"`
}

// NewMemberResponse maps a Member row to its API representation.
func NewMemberResponse(m *Member) MemberResponse {
	resp := MemberResponse{
		ID:                m.ID,
		Nickname:          m.Nickname,
		PreferredLanguage: m.PreferredLanguage,
	}
	if m.AvatarURL != nil {
		resp.AvatarURL = *m.AvatarURL
	}
	return resp
}

type VerifyInviteInput struct {
	Body struct {
		Code string `json:"code" minLength:"1" maxLength:"64" doc:"宿泊者招待コード"`
	}
}

type VerifyInviteOutput struct {
	Body struct {
		Valid bool `json:"valid" doc:"招待コードが正しければ true"`
	}
}

type SignupFormData struct {
	Nickname string        `form:"nickname" required:"true" minLength:"1" maxLength:"30" doc:"ニックネーム"`
	Avatar   huma.FormFile `form:"avatar" contentType:"image/jpeg,image/png,image/webp" doc:"プロフィール画像（任意）" required:"false"`
	Language string        `form:"language" default:"ja" enum:"ja,en,ko,zh" doc:"表示言語" required:"false"`
}

type SignupInput struct {
	RawBody huma.MultipartFormFiles[SignupFormData]
}

type SignupOutput struct {
	Body struct {
		Member       MemberResponse `json:"member" doc:"作成されたメンバー"`
		TransferCode string         `json:"transfer_code" doc:"引き継ぎコード（この1回のみ表示）"`
	}
}

type TransferInput struct {
	Body struct {
		TransferCode string `json:"transfer_code" minLength:"1" maxLength:"64" doc:"引き継ぎコード（XXXX-XXXX-XXXX）"`
	}
}

type TransferOutput struct {
	Body struct {
		Member MemberResponse `json:"member" doc:"復元されたメンバー"`
	}
}

type LogoutOutput struct {
	Body struct {
		OK bool `json:"ok" doc:"常に true"`
	}
}

type MeOutput struct {
	Body struct {
		Member MemberResponse `json:"member" doc:"ログイン中のメンバー"`
	}
}
