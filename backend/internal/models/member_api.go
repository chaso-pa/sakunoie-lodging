package models

import "github.com/danielgtaylor/huma/v2"

type UpdateProfileFormData struct {
	Nickname string        `form:"nickname" maxLength:"30" doc:"ニックネーム（省略時は変更しない）" required:"false"`
	Avatar   huma.FormFile `form:"avatar" contentType:"image/jpeg,image/png,image/webp" doc:"プロフィール画像（省略時は変更しない）" required:"false"`
	Language string        `form:"language" enum:"ja,en,ko,zh" doc:"表示言語（省略時は変更しない）" required:"false"`
}

type UpdateProfileInput struct {
	RawBody huma.MultipartFormFiles[UpdateProfileFormData]
}

type UpdateProfileOutput struct {
	Body struct {
		Member MemberResponse `json:"member" doc:"更新後のメンバー"`
	}
}
