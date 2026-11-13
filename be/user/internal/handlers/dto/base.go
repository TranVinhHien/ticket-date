package dto

import (
	"github.com/TranVinhHien/ticket-date/be/common/pagination"
	"github.com/TranVinhHien/ticket-date/be/common/response"
)

// Response aliases the standard response envelope from music-common.
type Response = response.AnyResponse

// BaseListDto is the response data shape for endpoints that return a list.
type BaseListDto struct {
	Limit int   `json:"limit"`
	Page  int   `json:"page"`
	Total int64 `json:"total"`
	List  any   `json:"list"`
}

// BaseQuery aliases the standard pagination query from music-common.
type BaseQuery = pagination.PaginationQuery
