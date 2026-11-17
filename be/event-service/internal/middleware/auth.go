package middleware

import (
	"errors"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	commonMiddleware "github.com/TranVinhHien/ticket-date/be/common/middleware"
	"github.com/TranVinhHien/ticket-date/be/common/response"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// UserContextMiddleware extracts user identity from Gateway headers or direct Bearer token.
func UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check Gateway injected headers
		if gatewayUserID := strings.TrimSpace(c.GetHeader(constants.HeaderUserID)); gatewayUserID != "" {
			c.Set(constants.CtxKeyUserID, gatewayUserID)
			if parsedUUID, err := uuid.Parse(gatewayUserID); err == nil {
				c.Set("user_uuid", parsedUUID)
			}
			if email := strings.TrimSpace(c.GetHeader(constants.HeaderUserEmail)); email != "" {
				c.Set(constants.CtxKeyUserEmail, email)
				c.Set(constants.CtxKeyEmail, email)
			}
			if rolesHeader := strings.TrimSpace(c.GetHeader(constants.HeaderUserRoles)); rolesHeader != "" {
				var roleList []string
				for _, r := range strings.Split(rolesHeader, ",") {
					if tr := strings.ToLower(strings.TrimSpace(r)); tr != "" {
						roleList = append(roleList, tr)
					}
				}
				c.Set(constants.CtxKeyUserRoles, roleList)
			}
			c.Next()
			return
		}

		// 2. Direct Bearer token validation if present
		authHeader := c.GetHeader(constants.HeaderAuthorization)
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == constants.BearerPrefix {
				tokenStr := parts[1]
				token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, errors.New("unexpected signing method")
					}
					return []byte(config.App.Token.Secret), nil
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(*JWTClaims); ok {
						c.Set(constants.CtxKeyUserID, claims.UserID.String())
						c.Set("user_uuid", claims.UserID)
						c.Set(constants.CtxKeyUserEmail, claims.Email)
						c.Set(constants.CtxKeyEmail, claims.Email)
					}
				}
			}
		}

		c.Next()
	}
}

// RequireAuth ensures that the request is authenticated.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := commonMiddleware.GetUserID(c)
		if !ok {
			response.Unauthorized(c, "Authentication required: missing or invalid user identity")
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetCurrentUser extracts CurrentUserContext from Gin context.
func GetCurrentUser(c *gin.Context) dto.CurrentUserContext {
	if uid, ok := commonMiddleware.GetUserID(c); ok {
		return dto.CurrentUserContext{UserID: &uid}
	}
	return dto.CurrentUserContext{UserID: nil}
}
