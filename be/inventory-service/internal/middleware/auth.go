package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderUserID = "X-User-ID"

// GatewayAuthMiddleware reads the X-User-ID header passed by the API Gateway.
func GatewayAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader(HeaderUserID)
		if userIDStr != "" {
			if id, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", id)
			}
		}
		c.Next()
	}
}

// GetUserID retrieves the parsed user UUID from the context if present.
func GetUserID(c *gin.Context) *uuid.UUID {
	val, exists := c.Get("user_id")
	if !exists {
		return nil
	}
	if id, ok := val.(uuid.UUID); ok {
		return &id
	}
	return nil
}
