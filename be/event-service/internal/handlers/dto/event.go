package dto

import (
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

type CreateTicketTypeInput struct {
	SKU         string                  `json:"sku" binding:"required"`
	Name        string                  `json:"name" binding:"required"`
	Description *string                 `json:"description"`
	Price       int64                   `json:"price" binding:"gte=0"`
	Currency    string                  `json:"currency"`
	MaxPerOrder int                     `json:"max_per_order"`
	SaleStartAt time.Time               `json:"sale_start_at" binding:"required"`
	SaleEndAt   time.Time               `json:"sale_end_at" binding:"required"`
	Attributes  models.JSONMap          `json:"attributes"`
	Status      models.TicketTypeStatus `json:"status"`
}

type UpdateTicketTypeInput struct {
	ID          *uuid.UUID               `json:"id"`
	SKU         *string                  `json:"sku"`
	Name        *string                  `json:"name"`
	Description *string                  `json:"description"`
	Price       *int64                   `json:"price"`
	Currency    *string                  `json:"currency"`
	MaxPerOrder *int                     `json:"max_per_order"`
	SaleStartAt *time.Time               `json:"sale_start_at"`
	SaleEndAt   *time.Time               `json:"sale_end_at"`
	Attributes  *models.JSONMap          `json:"attributes"`
	Status      *models.TicketTypeStatus `json:"status"`
	IsDeleted   bool                     `json:"is_deleted"`
}

type CreateEventRequest struct {
	OrganizerID uuid.UUID               `json:"organizer_id" binding:"required"`
	VenueID     uuid.UUID               `json:"venue_id" binding:"required"`
	Name        string                  `json:"name" binding:"required"`
	Slug        string                  `json:"slug"`
	Description *string                 `json:"description"`
	PosterURL   *string                 `json:"poster_url"`
	StartAt     time.Time               `json:"start_at" binding:"required"`
	EndAt       time.Time               `json:"end_at" binding:"required"`
	Status      models.EventStatus      `json:"status"`
	TicketTypes []CreateTicketTypeInput `json:"ticket_types"`
}

type UpdateEventRequest struct {
	OrganizerID *uuid.UUID              `json:"organizer_id"`
	VenueID     *uuid.UUID              `json:"venue_id"`
	Name        *string                 `json:"name"`
	Slug        *string                 `json:"slug"`
	Description *string                 `json:"description"`
	PosterURL   *string                 `json:"poster_url"`
	StartAt     *time.Time              `json:"start_at"`
	EndAt       *time.Time              `json:"end_at"`
	Status      *models.EventStatus     `json:"status"`
	TicketTypes []UpdateTicketTypeInput `json:"ticket_types,omitempty"`
}

type EventQuery struct {
	BaseQuery
	OrganizerID string `form:"organizer_id" json:"organizer_id"`
	VenueID     string `form:"venue_id" json:"venue_id"`
	Status      string `form:"status" json:"status"`
}
