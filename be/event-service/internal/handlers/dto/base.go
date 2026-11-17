package dto

import (
	"errors"

	"github.com/TranVinhHien/ticket-date/be/common/pagination"
	"github.com/TranVinhHien/ticket-date/be/common/response"
)

// Response aliases the standard response envelope from common.
type Response = response.AnyResponse

// BaseQuery aliases the standard pagination query from common.
type BaseQuery = pagination.PaginationQuery

var (
	ErrNotFound             = errors.New("resource not found")
	ErrOrganizerNotFound    = errors.New("organizer not found")
	ErrVenueNotFound        = errors.New("venue not found")
	ErrEventNotFound        = errors.New("event not found")
	ErrTicketTypeNotFound   = errors.New("ticket type not found")
	ErrSlugAlreadyExists    = errors.New("event slug already exists")
	ErrSKUAlreadyExists     = errors.New("ticket type SKU already exists")
	ErrInvalidPeriod        = errors.New("end time must be after start time")
	ErrInvalidSalePeriod    = errors.New("sale end time must be after sale start time")
	ErrInvalidCapacity      = errors.New("capacity must be greater than 0")
	ErrInvalidCoordinates   = errors.New("invalid latitude [-90, 90] or longitude [-180, 180]")
	ErrInvalidPrice         = errors.New("price must be greater than or equal to 0")
	ErrInvalidMaxPerOrder   = errors.New("max per order must be greater than 0")
	ErrOrganizerNameEmpty   = errors.New("organizer name cannot be empty")
	ErrVenueNameEmpty       = errors.New("venue name cannot be empty")
	ErrVenueAddressEmpty    = errors.New("venue address cannot be empty")
	ErrEventNameEmpty       = errors.New("event name cannot be empty")
	ErrTicketTypeNameEmpty  = errors.New("ticket type name cannot be empty")
)
