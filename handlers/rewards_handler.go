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

// RewardsHandler handles reward and redemption endpoints
type RewardsHandler struct {
	rewardService services.RewardService
}

// NewRewardsHandler creates a new rewards handler
func NewRewardsHandler(rewardService services.RewardService) *RewardsHandler {
	return &RewardsHandler{
		rewardService: rewardService,
	}
}

// HandleListRewards handles GET /v1/loyalty/rewards
func (h *RewardsHandler) HandleListRewards(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "GET /v1/loyalty/rewards")
	defer span.Finish()

	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// Parse query parameters
	query := r.URL.Query()
	req := &loyaltyv1.ListRewardsRequest{
		ActiveOnly: true, // Default to active only
	}

	// Parse active_only
	if activeOnlyStr := query.Get("active_only"); activeOnlyStr != "" {
		if activeOnly, err := strconv.ParseBool(activeOnlyStr); err == nil {
			req.ActiveOnly = activeOnly
		}
	}

	// Parse max_points
	if maxPointsStr := query.Get("max_points"); maxPointsStr != "" {
		if maxPoints, err := strconv.ParseInt(maxPointsStr, 10, 64); err == nil {
			req.MaxPoints = maxPoints
		}
	}

	// Call service (no authentication required for public catalog)
	response, err := h.rewardService.ListRewards(ctx, req)
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

// HandleRedeemReward handles POST /v1/loyalty/rewards/redeem
func (h *RewardsHandler) HandleRedeemReward(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "POST /v1/loyalty/rewards/redeem")
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
	var req loyaltyv1.RedeemRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ext.Error.Set(span, true)
		RespondWithError(w, Errors.InvalidRequest)
		return
	}

	// Call service
	response, err := h.rewardService.RedeemReward(ctx, &req, accountID)
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

// HandleListRedemptions handles GET /v1/loyalty/redemptions
func (h *RewardsHandler) HandleListRedemptions(w http.ResponseWriter, r *http.Request) {
	// Extract or start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "GET /v1/loyalty/redemptions")
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
	req := &loyaltyv1.ListRedemptionsRequest{
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

	// Parse status filter
	if statusStr := query.Get("status"); statusStr != "" {
		switch statusStr {
		case "ACTIVE":
			req.Status = loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE
		case "USED":
			req.Status = loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_USED
		case "REVERSED":
			req.Status = loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_REVERSED
		}
	}

	// Call service
	response, err := h.rewardService.ListRedemptions(ctx, accountID, req)
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

