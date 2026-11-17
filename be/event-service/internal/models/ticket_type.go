package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketTypeStatus string

const (
	TicketTypeStatusDraft  TicketTypeStatus = "DRAFT"
	TicketTypeStatusActive TicketTypeStatus = "ACTIVE"
	TicketTypeStatusPaused TicketTypeStatus = "PAUSED"
	TicketTypeStatusEnded  TicketTypeStatus = "ENDED"
)

type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return "{}", nil
	}
	return string(b), nil
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to unmarshal JSONMap value")
	}
	return json.Unmarshal(bytes, j)
}

type TicketType struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EventID     uuid.UUID        `gorm:"type:uuid;not null;index:idx_ticket_types_event" json:"event_id"`
	SKU         string           `gorm:"type:varchar(100);uniqueIndex;not null" json:"sku"`
	Name        string           `gorm:"type:varchar(150);not null" json:"name"`
	Description *string          `gorm:"type:text" json:"description,omitempty"`
	Price       int64            `gorm:"type:bigint;not null" json:"price"`
	Currency    string           `gorm:"type:varchar(3);not null;default:'VND'" json:"currency"`
	MaxPerOrder int              `gorm:"type:int;not null;default:4" json:"max_per_order"`
	SaleStartAt time.Time        `gorm:"type:timestamptz;not null;index:idx_ticket_types_status_sale" json:"sale_start_at"`
	SaleEndAt   time.Time        `gorm:"type:timestamptz;not null;index:idx_ticket_types_status_sale" json:"sale_end_at"`
	Attributes  JSONMap          `gorm:"type:jsonb;not null;default:'{}'" json:"attributes"`
	Status      TicketTypeStatus `gorm:"type:varchar(30);not null;default:'DRAFT';index:idx_ticket_types_status_sale" json:"status"`

	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedBy *uuid.UUID     `gorm:"type:uuid" json:"deleted_by,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Event *Event `gorm:"foreignKey:EventID" json:"event,omitempty"`
}

func (TicketType) TableName() string {
	return "ticket_types"
}

func (t *TicketType) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Currency == "" {
		t.Currency = "VND"
	}
	if t.MaxPerOrder <= 0 {
		t.MaxPerOrder = 4
	}
	if t.Status == "" {
		t.Status = TicketTypeStatusDraft
	}
	if t.Attributes == nil {
		t.Attributes = JSONMap{}
	}
	return nil
}
