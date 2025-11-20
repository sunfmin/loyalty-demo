package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for storing authenticated user ID
	UserIDKey contextKey = "user_id"
	// RolesKey is the context key for storing user roles
	RolesKey contextKey = "roles"
)

// Authentication middleware extracts and validates JWT tokens
// For now, this is a simplified implementation for testing
// In production, this would validate actual JWT tokens
func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Validate Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		if token == "" {
			http.Error(w, `{"error":"Missing token"}`, http.StatusUnauthorized)
			return
		}

		// TODO: Validate JWT token and extract claims
		// For now, use a simplified mock implementation for development
		// In production, this would use a proper JWT library like golang-jwt/jwt
		
		// Mock user ID extraction (in production, extract from validated JWT claims)
		userID := extractUserIDFromToken(token)
		if userID == "" {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Mock roles extraction (in production, extract from validated JWT claims)
		roles := extractRolesFromToken(token)

		// Add user ID and roles to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, RolesKey, roles)

		// Call next handler with authenticated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractUserIDFromToken is a mock implementation for development
// In production, this would validate and parse a real JWT token
func extractUserIDFromToken(token string) string {
	// Mock implementation: use token as user ID for testing
	// In production, decode JWT and extract 'sub' claim
	if token == "test-token" {
		return "test-user-123"
	}
	// For any other token, return it as-is for development
	return token
}

// extractRolesFromToken is a mock implementation for development
// In production, this would extract roles from JWT claims
func extractRolesFromToken(token string) []string {
	// Mock implementation: check for admin token
	if strings.Contains(token, "admin") {
		return []string{"user", "admin"}
	}
	return []string{"user"}
}

// GetUserID extracts the authenticated user ID from context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetRoles extracts the user roles from context
func GetRoles(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(RolesKey).([]string)
	return roles, ok
}

// HasRole checks if the user has a specific role
func HasRole(ctx context.Context, role string) bool {
	roles, ok := GetRoles(ctx)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

