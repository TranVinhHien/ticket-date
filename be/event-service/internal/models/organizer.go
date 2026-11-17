package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizerStatus string

const (
	OrganizerStatusActive   OrganizerStatus = "ACTIVE"
	OrganizerStatusInactive OrganizerStatus = "INACTIVE"
)

type Organizer struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationName string          `gorm:"type:varchar(255);not null" json:"organization_name"`
	Description      *string         `gorm:"type:text" json:"description,omitempty"`
	LogoURL          *string         `gorm:"type:text" json:"logo_url,omitempty"`
	TaxCode          *string         `gorm:"type:varchar(50)" json:"tax_code,omitempty"`
	Address          *string         `gorm:"type:text" json:"address,omitempty"`
	Status           OrganizerStatus `gorm:"type:varchar(30);not null;default:'ACTIVE'" json:"status"`

	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedBy *uuid.UUID     `gorm:"type:uuid" json:"deleted_by,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Venues []Venue `gorm:"foreignKey:OrganizerID;constraint:OnDelete:CASCADE;" json:"venues,omitempty"`
	Events []Event `gorm:"foreignKey:OrganizerID;constraint:OnDelete:CASCADE;" json:"events,omitempty"`
}

func (Organizer) TableName() string {
	return "organizers"
}

func (o *Organizer) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.Status == "" {
		o.Status = OrganizerStatusActive
	}
	return nil
}
