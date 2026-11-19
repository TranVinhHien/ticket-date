package grpcContext

import (
	"context"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// Standard gRPC metadata keys (lowercase according to HTTP/2 and gRPC specs).
const (
	MDKeyRequestID = "x-request-id"
	MDKeyUserID    = "x-user-id"
	MDKeyUserEmail = "x-user-email"
	MDKeyUserRoles = "x-user-roles"
)

// FromGinContext extracts identity and request tracking information from a Gin context
// and creates a new outgoing gRPC context with the corresponding metadata.
func FromGinContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	md := metadata.Pairs()

	// 1. Request ID
	reqID := c.GetString(constants.CtxKeyRequestID)
	if reqID == "" {
		reqID = c.GetHeader(constants.HeaderRequestID)
	}
	if reqID != "" {
		md.Set(MDKeyRequestID, reqID)
	}

	// 2. User ID
	userID := c.GetString(constants.CtxKeyUserID)
	if userID == "" {
		userID = c.GetHeader(constants.HeaderUserID)
	}
	if userID != "" {
		md.Set(MDKeyUserID, userID)
	}

	// 3. User Email
	userEmail := c.GetString(constants.CtxKeyUserEmail)
	if userEmail == "" {
		userEmail = c.GetHeader(constants.HeaderUserEmail)
	}
	if userEmail != "" {
		md.Set(MDKeyUserEmail, userEmail)
	}

	// 4. User Roles
	var roles []string
	if val, exists := c.Get(constants.CtxKeyUserRoles); exists {
		if r, ok := val.([]string); ok {
			roles = r
		}
	}
	if len(roles) == 0 {
		rawRoles := c.GetHeader(constants.HeaderUserRoles)
		if rawRoles != "" {
			for _, r := range strings.Split(rawRoles, ",") {
				trimmed := strings.TrimSpace(r)
				if trimmed != "" {
					roles = append(roles, trimmed)
				}
			}
		}
	}
	if len(roles) > 0 {
		md.Set(MDKeyUserRoles, strings.Join(roles, ","))
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// WithOutgoingMetadata attaches key-value pairs to the outgoing gRPC context.
func WithOutgoingMetadata(ctx context.Context, kv ...string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.Pairs(kv...)
	} else {
		md = md.Copy()
		for i := 0; i < len(kv)-1; i += 2 {
			md.Set(strings.ToLower(kv[i]), kv[i+1])
		}
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// PropagateContextToOutgoing copies standard identity and trace values from Go context
// (CtxKeyRequestID, CtxKeyUserID, CtxKeyUserEmail, CtxKeyUserRoles) into outgoing gRPC metadata.
func PropagateContextToOutgoing(ctx context.Context) context.Context {
	pairs := make([]string, 0, 8)

	if reqID := GetRequestID(ctx); reqID != "" {
		pairs = append(pairs, MDKeyRequestID, reqID)
	}

	if uidStr := GetUserIDString(ctx); uidStr != "" {
		pairs = append(pairs, MDKeyUserID, uidStr)
	}

	if email := GetUserEmail(ctx); email != "" {
		pairs = append(pairs, MDKeyUserEmail, email)
	}

	if roles := GetUserRoles(ctx); len(roles) > 0 {
		pairs = append(pairs, MDKeyUserRoles, strings.Join(roles, ","))
	}

	if len(pairs) == 0 {
		return ctx
	}

	return WithOutgoingMetadata(ctx, pairs...)
}

// ToOutgoingContext transfers specific keys from Go context.Context into gRPC outgoing metadata.
func ToOutgoingContext(ctx context.Context, keys ...string) context.Context {
	pairs := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		val := ctx.Value(k)
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case string:
			pairs = append(pairs, strings.ToLower(k), v)
		case []string:
			pairs = append(pairs, strings.ToLower(k), strings.Join(v, ","))
		case uuid.UUID:
			pairs = append(pairs, strings.ToLower(k), v.String())
		}
	}

	if len(pairs) == 0 {
		return ctx
	}
	return WithOutgoingMetadata(ctx, pairs...)
}

// GetRequestID extracts the Request ID from gRPC incoming metadata or Go context.
func GetRequestID(ctx context.Context) string {
	if val := ctx.Value(constants.CtxKeyRequestID); val != nil {
		if id, ok := val.(string); ok && id != "" {
			return id
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(MDKeyRequestID); len(values) > 0 && values[0] != "" {
			return values[0]
		}
		if values := md.Get("request_id"); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}

// GetUserID extracts the user UUID from gRPC incoming metadata or Go context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	if val := ctx.Value(constants.CtxKeyUserID); val != nil {
		if uid, ok := val.(uuid.UUID); ok {
			return uid, true
		}
		if uidStr, ok := val.(string); ok {
			if parsed, err := uuid.Parse(uidStr); err == nil {
				return parsed, true
			}
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(MDKeyUserID); len(values) > 0 && values[0] != "" {
			if parsed, err := uuid.Parse(values[0]); err == nil {
				return parsed, true
			}
		}
	}

	return uuid.Nil, false
}

// GetUserIDString extracts the user ID as string from gRPC incoming metadata or Go context,
// regardless of whether it conforms to UUID formatting.
func GetUserIDString(ctx context.Context) string {
	if uid, ok := GetUserID(ctx); ok {
		return uid.String()
	}

	if val := ctx.Value(constants.CtxKeyUserID); val != nil {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(MDKeyUserID); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}

// GetUserEmail extracts the user email from gRPC incoming metadata or Go context.
func GetUserEmail(ctx context.Context) string {
	if val := ctx.Value(constants.CtxKeyUserEmail); val != nil {
		if email, ok := val.(string); ok {
			return email
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(MDKeyUserEmail); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}

// GetUserRoles extracts user roles from gRPC incoming metadata or Go context.
func GetUserRoles(ctx context.Context) []string {
	if val := ctx.Value(constants.CtxKeyUserRoles); val != nil {
		if roles, ok := val.([]string); ok {
			return roles
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(MDKeyUserRoles); len(values) > 0 && values[0] != "" {
			raw := values[0]
			var roles []string
			for _, r := range strings.Split(raw, ",") {
				trimmed := strings.TrimSpace(r)
				if trimmed != "" {
					roles = append(roles, trimmed)
				}
			}
			return roles
		}
	}

	return []string{}
}
