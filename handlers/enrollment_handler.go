package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/services"
)

// EnrollmentHandler handles loyalty enrollment endpoints
type EnrollmentHandler struct {
	loyaltyService services.LoyaltyService
}

// NewEnrollmentHandler creates a new enrollment handler
func NewEnrollmentHandler(loyaltyService services.LoyaltyService) *EnrollmentHandler {
	return &EnrollmentHandler{
		loyaltyService: loyaltyService,
	}
}

// HandleEnroll handles POST /v1/loyalty/enroll
func (h *EnrollmentHandler) HandleEnroll(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "POST /v1/loyalty/enroll")
	defer span.Finish()

	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// Extract account ID from context (set by auth middleware)
	accountID, ok := middleware.GetUserID(ctx)
	if !ok {
		ext.Error.Set(span, true)
		ext.HTTPStatusCode.Set(span, uint16(http.StatusUnauthorized))
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Unauthorized - missing user authentication"}`))
		return
	}

	span.SetTag("account_id", accountID)

	// Parse request body
	var req loyaltyv1.EnrollCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ext.Error.Set(span, true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	customer, err := h.loyaltyService.EnrollCustomer(ctx, &req, accountID)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Success response
	ext.HTTPStatusCode.Set(span, uint16(http.StatusCreated))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	
	response := &loyaltyv1.EnrollCustomerResponse{
		Customer: customer,
	}
	json.NewEncoder(w).Encode(response)
}

// HandleGetStatus handles GET /v1/loyalty/me
func (h *EnrollmentHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "GET /v1/loyalty/me")
	defer span.Finish()

	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// Extract account ID from context (set by auth middleware)
	accountID, ok := middleware.GetUserID(ctx)
	if !ok {
		ext.Error.Set(span, true)
		ext.HTTPStatusCode.Set(span, uint16(http.StatusUnauthorized))
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Unauthorized - missing user authentication"}`))
		return
	}

	span.SetTag("account_id", accountID)

	// Call service
	response, err := h.loyaltyService.GetCustomerStatus(ctx, accountID)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Success response
	ext.HTTPStatusCode.Set(span, uint16(http.StatusOK))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleListTiers handles GET /v1/loyalty/tiers
func (h *EnrollmentHandler) HandleListTiers(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "GET /v1/loyalty/tiers")
	defer span.Finish()

	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// Call service (no authentication required for public tier list)
	response, err := h.loyaltyService.ListTiers(ctx)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Success response
	ext.HTTPStatusCode.Set(span, uint16(http.StatusOK))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

