package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/TranVinhHien/ticket-date/be/common/pagination"
	"github.com/TranVinhHien/ticket-date/be/common/response"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/gin-gonic/gin"
)

func respond(c *gin.Context, status int, message string, data interface{}) {
	response.Send(c, status, strconv.Itoa(status), message, nil, data)
}

func respondPaginated[T any](c *gin.Context, list []T, meta pagination.PaginationMeta) {
	response.Paginated(c, list, meta)
}

func respondError(c *gin.Context, err error) {
	status := mapErrorToStatus(err)
	response.Error(c, status, err)
}

func mapErrorToStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if errors.Is(err, dto.ErrNotFound) ||
		errors.Is(err, dto.ErrOrganizerNotFound) ||
		errors.Is(err, dto.ErrVenueNotFound) ||
		errors.Is(err, dto.ErrEventNotFound) ||
		errors.Is(err, dto.ErrTicketTypeNotFound) {
		return http.StatusNotFound
	}

	if errors.Is(err, dto.ErrSlugAlreadyExists) ||
		errors.Is(err, dto.ErrSKUAlreadyExists) {
		return http.StatusConflict
	}

	if errors.Is(err, dto.ErrInvalidPeriod) ||
		errors.Is(err, dto.ErrInvalidSalePeriod) ||
		errors.Is(err, dto.ErrInvalidCapacity) ||
		errors.Is(err, dto.ErrInvalidCoordinates) ||
		errors.Is(err, dto.ErrInvalidPrice) ||
		errors.Is(err, dto.ErrInvalidMaxPerOrder) ||
		errors.Is(err, dto.ErrOrganizerNameEmpty) ||
		errors.Is(err, dto.ErrVenueNameEmpty) ||
		errors.Is(err, dto.ErrVenueAddressEmpty) ||
		errors.Is(err, dto.ErrEventNameEmpty) ||
		errors.Is(err, dto.ErrTicketTypeNameEmpty) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}
