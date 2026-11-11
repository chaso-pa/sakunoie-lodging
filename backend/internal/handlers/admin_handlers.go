package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

// AdminHandlers serve /api/v1/admin. The AdminScope comes from the
// AdminAuth middleware, never from request parameters.
type AdminHandlers struct {
	Registrations *services.AdminRegistrationService
}

func (h *AdminHandlers) ListAdminFacilities(ctx context.Context, _ *models.ListAdminFacilitiesInput) (*models.ListAdminFacilitiesOutput, error) {
	facs, err := h.Registrations.Facilities(ctx, middlewares.AdminScopeFrom(ctx))
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	out := &models.ListAdminFacilitiesOutput{}
	out.Body.Facilities = facs
	return out, nil
}

func (h *AdminHandlers) ListAdminRegistrations(ctx context.Context, in *models.ListAdminRegistrationsInput) (*models.ListAdminRegistrationsOutput, error) {
	list, err := h.Registrations.List(ctx, middlewares.AdminScopeFrom(ctx), services.AdminListFilter{
		FacilityID: in.FacilityID, From: in.From, To: in.To, Q: in.Q, Page: in.Page,
	})
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.ListAdminRegistrationsOutput{Body: *list}, nil
}

func adminDetailOutput(d *models.AdminRegistrationDetail) *models.AdminRegistrationOutput {
	return &models.AdminRegistrationOutput{ETag: `"` + d.Version + `"`, Body: *d}
}

func (h *AdminHandlers) GetAdminRegistration(ctx context.Context, in *models.GetAdminRegistrationInput) (*models.AdminRegistrationOutput, error) {
	d, err := h.Registrations.Detail(ctx, middlewares.AdminScopeFrom(ctx), in.ID, in.Reveal)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return adminDetailOutput(d), nil
}

func (h *AdminHandlers) PatchAdminRegistration(ctx context.Context, in *models.PatchAdminRegistrationInput) (*models.AdminRegistrationOutput, error) {
	d, err := h.Registrations.Patch(ctx, middlewares.AdminScopeFrom(ctx), in.ID, in.IfMatch, &in.Body)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return adminDetailOutput(d), nil
}

func (h *AdminHandlers) VerifyIdentity(ctx context.Context, in *models.AdminRegistrationActionInput) (*models.AdminRegistrationOutput, error) {
	d, err := h.Registrations.VerifyIdentity(ctx, middlewares.AdminScopeFrom(ctx), in.ID)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return adminDetailOutput(d), nil
}

func (h *AdminHandlers) ReissueEditLink(ctx context.Context, in *models.AdminRegistrationActionInput) (*models.ReissueEditLinkOutput, error) {
	res, err := h.Registrations.ReissueEditLink(ctx, middlewares.AdminScopeFrom(ctx), in.ID)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.ReissueEditLinkOutput{Body: *res}, nil
}

func (h *AdminHandlers) ExportAdminRegistrations(ctx context.Context, in *models.ExportAdminRegistrationsInput) (*models.ExportAdminRegistrationsOutput, error) {
	res, err := h.Registrations.Export(ctx, middlewares.AdminScopeFrom(ctx), services.ExportFilter{
		FacilityID: in.FacilityID, From: in.From, To: in.To,
	})
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.ExportAdminRegistrationsOutput{
		ContentType:        "text/csv; charset=utf-8",
		ContentDisposition: `attachment; filename="` + res.Filename + `"`,
		Body:               res.CSV,
	}, nil
}
