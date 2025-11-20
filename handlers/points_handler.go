package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/services"
)

// PointsHandler handles point earning and transaction endpoints
type PointsHandler struct {
	transactionService services.TransactionService
	loyaltyService     services.LoyaltyService // For tier evaluation after earning
}

// NewPointsHandler creates a new points handler
func NewPointsHandler(transactionService services.TransactionService, loyaltyService services.LoyaltyService) *PointsHandler {
	return &PointsHandler{
		transactionService: transactionService,
		loyaltyService:     loyaltyService,
	}
}

// HandleEarnPoints handles POST /v1/loyalty/points/earn
func (h *PointsHandler) HandleEarnPoints(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "POST /v1/loyalty/points/earn")
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
	var req loyaltyv1.EarnPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ext.Error.Set(span, true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	response, err := h.transactionService.EarnPoints(ctx, &req, accountID)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		HandleServiceError(w, err)
		return
	}

	// Evaluate tier progression after earning points (best effort - don't fail if tier evaluation fails)
	if h.loyaltyService != nil && response.Transaction != nil {
		customerID := response.Transaction.CustomerId
		if _, tierErr := h.loyaltyService.EvaluateCustomerTier(ctx, customerID); tierErr != nil {
			// Log but don't fail the request
			span.SetTag("tier_evaluation_error", tierErr.Error())
		}
	}

	// Success response
	ext.HTTPStatusCode.Set(span, uint16(http.StatusCreated))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleListTransactions handles GET /v1/loyalty/transactions
func (h *PointsHandler) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "GET /v1/loyalty/transactions")
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

	// Parse query parameters
	query := r.URL.Query()
	req := &loyaltyv1.ListTransactionsRequest{
		Limit:  50,
		Offset: 0,
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := parseInt32(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := parseInt32(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	// Parse type filter
	if typeStr := query.Get("type"); typeStr != "" {
		switch typeStr {
		case "EARN":
			req.Type = loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN
		case "REDEMPTION":
			req.Type = loyaltyv1.TransactionType_TRANSACTION_TYPE_REDEMPTION
		case "ADJUSTMENT":
			req.Type = loyaltyv1.TransactionType_TRANSACTION_TYPE_ADJUSTMENT
		case "REFERRAL":
			req.Type = loyaltyv1.TransactionType_TRANSACTION_TYPE_REFERRAL
		case "EXPIRATION":
			req.Type = loyaltyv1.TransactionType_TRANSACTION_TYPE_EXPIRATION
		}
	}

	// Validate limit
	if req.Limit > 100 {
		ext.Error.Set(span, true)
		RespondWithError(w, Errors.ValueOutOfRange)
		return
	}

	// Call service
	response, err := h.transactionService.ListTransactions(ctx, accountID, req)
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

// Helper function to parse int32 from string
func parseInt32(s string) (int32, error) {
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

