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

// PointsHandler handles point earning and transaction endpoints
type PointsHandler struct {
	transactionService services.TransactionService
}

// NewPointsHandler creates a new points handler
func NewPointsHandler(transactionService services.TransactionService) *PointsHandler {
	return &PointsHandler{
		transactionService: transactionService,
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
	// TODO: Parse query parameters into request object
	req := &loyaltyv1.ListTransactionsRequest{
		Limit:  50,
		Offset: 0,
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

