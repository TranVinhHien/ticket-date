package dto

import (
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

type CreateOrganizerRequest struct {
	OrganizationName string                 `json:"organization_name" binding:"required"`
	Description      *string                `json:"description"`
	LogoURL          *string                `json:"logo_url"`
	TaxCode          *string                `json:"tax_code"`
	Address          *string                `json:"address"`
	Status           models.OrganizerStatus `json:"status"`
}

type UpdateOrganizerRequest struct {
	OrganizationName *string                 `json:"organization_name"`
	Description      *string                 `json:"description"`
	LogoURL          *string                 `json:"logo_url"`
	TaxCode          *string                 `json:"tax_code"`
	Address          *string                 `json:"address"`
	Status           *models.OrganizerStatus `json:"status"`
}

type OrganizerQuery struct {
	BaseQuery
	Status string `form:"status" json:"status"`
}

type OrganizerDetailResponse struct {
	Organizer *models.Organizer `json:"organizer"`
}

type OrganizerListResponse struct {
	List       []*models.Organizer `json:"list"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

type CurrentUserContext struct {
	UserID *uuid.UUID
}
