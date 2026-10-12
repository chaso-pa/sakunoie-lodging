package handlers

import (
	"context"
	"io"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
)

type MemberHandlers struct {
	members *services.MemberService
}

func NewMemberHandlers(members *services.MemberService) *MemberHandlers {
	return &MemberHandlers{members: members}
}

// UpdateProfile changes the logged-in member's own profile.
func (h *MemberHandlers) UpdateProfile(ctx context.Context, input *models.UpdateProfileInput) (*models.UpdateProfileOutput, error) {
	data := input.RawBody.Data()
	var avatar io.Reader
	if data.Avatar.IsSet {
		defer data.Avatar.Close()
		avatar = data.Avatar
	}
	member, err := h.members.UpdateProfile(ctx, middlewares.GetMemberID(ctx), data.Nickname, data.Language, avatar)
	if err != nil {
		return nil, serviceError(ctx, "update profile", err)
	}
	resp := &models.UpdateProfileOutput{}
	resp.Body.Member = models.NewMemberResponse(member)
	return resp, nil
}
