package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventStatus string

const (
	EventStatusDraft     EventStatus = "DRAFT"
	EventStatusPublished EventStatus = "PUBLISHED"
	EventStatusCancelled EventStatus = "CANCELLED"
	EventStatusCompleted EventStatus = "COMPLETED"
)

type Event struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizerID uuid.UUID   `gorm:"type:uuid;not null;index:idx_events_organizer" json:"organizer_id"`
	VenueID     uuid.UUID   `gorm:"type:uuid;not null;index:idx_events_venue" json:"venue_id"`
	Name        string      `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string      `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Description *string     `gorm:"type:text" json:"description,omitempty"`
	PosterURL   *string     `gorm:"type:text" json:"poster_url,omitempty"`
	StartAt     time.Time   `gorm:"type:timestamptz;not null;index:idx_events_status_start" json:"start_at"`
	EndAt       time.Time   `gorm:"type:timestamptz;not null" json:"end_at"`
	Status      EventStatus `gorm:"type:varchar(30);not null;default:'DRAFT';index:idx_events_status_start" json:"status"`

	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedBy *uuid.UUID     `gorm:"type:uuid" json:"deleted_by,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Organizer   *Organizer   `gorm:"foreignKey:OrganizerID" json:"organizer,omitempty"`
	Venue       *Venue       `gorm:"foreignKey:VenueID" json:"venue,omitempty"`
	TicketTypes []TicketType `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE;" json:"ticket_types,omitempty"`
}

func (Event) TableName() string {
	return "events"
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Status == "" {
		e.Status = EventStatusDraft
	}
	return nil
}
