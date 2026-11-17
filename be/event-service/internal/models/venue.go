package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VenueStatus string

const (
	VenueStatusActive    VenueStatus = "ACTIVE"
	VenueStatusInactive  VenueStatus = "INACTIVE"
	VenueStatusSuspended VenueStatus = "SUSPENDED"
)

type Venue struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizerID uuid.UUID   `gorm:"type:uuid;not null;index:idx_venues_organizer" json:"organizer_id"`
	Name        string      `gorm:"type:varchar(255);not null" json:"name"`
	Address     string      `gorm:"type:text;not null" json:"address"`
	City        *string     `gorm:"type:varchar(100)" json:"city,omitempty"`
	Country     string      `gorm:"type:varchar(100);not null;default:'Vietnam'" json:"country"`
	MapURL      *string     `gorm:"type:text" json:"map_url,omitempty"`
	Latitude    *float64    `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	Longitude   *float64    `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Capacity    int         `gorm:"not null" json:"capacity"`
	Description *string     `gorm:"type:text" json:"description,omitempty"`
	Status      VenueStatus `gorm:"type:varchar(30);not null;default:'ACTIVE'" json:"status"`

	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedBy *uuid.UUID     `gorm:"type:uuid" json:"deleted_by,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Organizer *Organizer `gorm:"foreignKey:OrganizerID" json:"organizer,omitempty"`
	Events    []Event    `gorm:"foreignKey:VenueID" json:"events,omitempty"`
}

func (Venue) TableName() string {
	return "venues"
}

func (v *Venue) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.Country == "" {
		v.Country = "Vietnam"
	}
	if v.Status == "" {
		v.Status = VenueStatusActive
	}
	return nil
}
