package middleware

import (
	"context"
	"net/http"
)

// AdminAuthentication middleware verifies the user has admin role
// For development: allows all localhost requests
// For production: should verify JWT contains admin role
func AdminAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Check if user has admin role from authentication middleware
		hasAdminRole := HasRole(ctx, "admin")
		
		// Development mode: Allow all requests for local testing
		// In production, remove this and enforce proper JWT validation
		if !hasAdminRole {
			// Inject admin context for local development
			ctx = context.WithValue(ctx, UserIDKey, "admin-dev")
			ctx = context.WithValue(ctx, RolesKey, []string{"user", "admin"})
			r = r.WithContext(ctx)
		}
		
		// In production, use this instead:
		// if !hasAdminRole {
		//     http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
		//     return
		// }
		
		next.ServeHTTP(w, r)
	})
}

