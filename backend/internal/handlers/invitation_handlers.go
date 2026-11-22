package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

// InvitationHandlers serve the admin invitation console and the public
// invitation check of the registration wizard.
type InvitationHandlers struct {
	Invitations *services.InvitationService
}

func (h *InvitationHandlers) CreateAdminInvitation(ctx context.Context, in *models.CreateAdminInvitationInput) (*models.CreateAdminInvitationOutput, error) {
	res, err := h.Invitations.Issue(ctx, middlewares.AdminScopeFrom(ctx), services.AdminInvitationInput{
		FacilityID: in.Body.FacilityID, CheckInDate: in.Body.CheckInDate, CheckOutDate: in.Body.CheckOutDate,
		BookingReference: in.Body.BookingReference, Note: in.Body.Note,
	})
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.CreateAdminInvitationOutput{Body: *res}, nil
}

func (h *InvitationHandlers) ListAdminInvitations(ctx context.Context, in *models.ListAdminInvitationsInput) (*models.ListAdminInvitationsOutput, error) {
	list, err := h.Invitations.List(ctx, middlewares.AdminScopeFrom(ctx), services.InvitationListFilter{
		FacilityID: in.FacilityID, Status: in.Status, Page: in.Page,
	})
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.ListAdminInvitationsOutput{Body: *list}, nil
}

func (h *InvitationHandlers) RevokeAdminInvitation(ctx context.Context, in *models.RevokeAdminInvitationInput) (*models.RevokeAdminInvitationOutput, error) {
	inv, err := h.Invitations.Revoke(ctx, middlewares.AdminScopeFrom(ctx), in.ID)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.RevokeAdminInvitationOutput{Body: *inv}, nil
}

func (h *InvitationHandlers) GetCurrentInvitation(ctx context.Context, _ *models.GetCurrentInvitationInput) (*models.GetCurrentInvitationOutput, error) {
	cur, err := h.Invitations.Current(ctx, middlewares.InvitationTokenFrom(ctx))
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.GetCurrentInvitationOutput{CacheControl: "no-store", Body: *cur}, nil
}
