package dto

import (
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

type CreateVenueRequest struct {
	OrganizerID uuid.UUID          `json:"organizer_id" binding:"required"`
	Name        string             `json:"name" binding:"required"`
	Address     string             `json:"address" binding:"required"`
	City        *string            `json:"city"`
	Country     string             `json:"country"`
	MapURL      *string            `json:"map_url"`
	Latitude    *float64           `json:"latitude"`
	Longitude   *float64           `json:"longitude"`
	Capacity    int                `json:"capacity" binding:"required,gt=0"`
	Description *string            `json:"description"`
	Status      models.VenueStatus `json:"status"`
}

type UpdateVenueRequest struct {
	OrganizerID *uuid.UUID          `json:"organizer_id"`
	Name        *string             `json:"name"`
	Address     *string             `json:"address"`
	City        *string             `json:"city"`
	Country     *string             `json:"country"`
	MapURL      *string             `json:"map_url"`
	Latitude    *float64            `json:"latitude"`
	Longitude   *float64            `json:"longitude"`
	Capacity    *int                `json:"capacity"`
	Description *string             `json:"description"`
	Status      *models.VenueStatus `json:"status"`
}

type VenueQuery struct {
	BaseQuery
	OrganizerID string `form:"organizer_id" json:"organizer_id"`
	City        string `form:"city" json:"city"`
	Status      string `form:"status" json:"status"`
}
