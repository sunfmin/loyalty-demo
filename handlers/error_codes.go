package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	
	"github.com/yourorg/loyalty-demo/services"
)

// ErrorCode represents an HTTP error with code, message, status, and optional service error mapping
type ErrorCode struct {
	Code       string // Machine-readable error code
	Message    string // Human-readable error message
	HTTPStatus int    // HTTP status code
	ServiceErr error  // Optional: Maps to service sentinel error for automatic checking
}

// Errors is a singleton instance with all HTTP error definitions
var Errors = struct {
	// Validation errors (400)
	InvalidRequest   ErrorCode
	ValidationFailed ErrorCode
	MissingRequired  ErrorCode
	ValueOutOfRange  ErrorCode
	InvalidType      ErrorCode
	InsufficientBalance ErrorCode
	
	// Not found errors (404)
	CustomerNotFound  ErrorCode
	RewardNotFound    ErrorCode
	RedemptionNotFound ErrorCode
	CampaignNotFound  ErrorCode
	TemplateNotFound  ErrorCode
	
	// Conflict errors (409)
	AlreadyEnrolled  ErrorCode
	RedemptionAlreadyReversed ErrorCode
	AlreadyExists    ErrorCode
	
	// Internal errors (500)
	InternalError ErrorCode
}{
	// Validation errors - mapped to service errors
	InvalidRequest:   ErrorCode{"INVALID_REQUEST", "Invalid request body", http.StatusBadRequest, services.ErrInvalidRequest},
	MissingRequired:  ErrorCode{"MISSING_REQUIRED", "Required field missing", http.StatusBadRequest, services.ErrMissingRequired},
	ValueOutOfRange:  ErrorCode{"VALUE_OUT_OF_RANGE", "Value out of range", http.StatusBadRequest, services.ErrValueOutOfRange},
	InvalidType:      ErrorCode{"INVALID_TYPE", "Invalid type", http.StatusBadRequest, services.ErrInvalidType},
	InsufficientBalance: ErrorCode{"INSUFFICIENT_BALANCE", "Insufficient point balance", http.StatusBadRequest, services.ErrInsufficientBalance},
	
	// Not found errors - mapped to service errors
	CustomerNotFound:  ErrorCode{"CUSTOMER_NOT_FOUND", "Customer not enrolled", http.StatusNotFound, services.ErrNotEnrolled},
	RewardNotFound:    ErrorCode{"REWARD_NOT_FOUND", "Reward not found", http.StatusNotFound, services.ErrRewardNotFound},
	RedemptionNotFound: ErrorCode{"REDEMPTION_NOT_FOUND", "Redemption not found", http.StatusNotFound, services.ErrRedemptionNotFound},
	CampaignNotFound:  ErrorCode{"CAMPAIGN_NOT_FOUND", "Campaign not found", http.StatusNotFound, services.ErrCampaignNotFound},
	TemplateNotFound:  ErrorCode{"TEMPLATE_NOT_FOUND", "Template not found", http.StatusNotFound, services.ErrTemplateNotFound},
	
	// Conflict errors - mapped to service errors
	AlreadyEnrolled:  ErrorCode{"ALREADY_ENROLLED", "Customer already enrolled", http.StatusConflict, services.ErrAlreadyEnrolled},
	RedemptionAlreadyReversed: ErrorCode{"REDEMPTION_ALREADY_REVERSED", "Redemption already reversed", http.StatusConflict, services.ErrRedemptionAlreadyReversed},
	AlreadyExists:    ErrorCode{"ALREADY_EXISTS", "Resource already exists", http.StatusConflict, services.ErrAlreadyExists},
	
	// Internal errors - no mapping (catch-all)
	InternalError:    ErrorCode{"INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError, nil},
}

// AllErrors returns a slice of all error codes for iteration
func AllErrors() []ErrorCode {
	return []ErrorCode{
		Errors.InvalidRequest,
		Errors.MissingRequired,
		Errors.ValueOutOfRange,
		Errors.InvalidType,
		Errors.InsufficientBalance,
		Errors.CustomerNotFound,
		Errors.RewardNotFound,
		Errors.RedemptionNotFound,
		Errors.CampaignNotFound,
		Errors.TemplateNotFound,
		Errors.AlreadyEnrolled,
		Errors.RedemptionAlreadyReversed,
		Errors.AlreadyExists,
		Errors.InternalError,
	}
}

// ErrorResponse represents the JSON structure for error responses
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// RespondWithError writes an error response with the given ErrorCode
func RespondWithError(w http.ResponseWriter, errCode ErrorCode) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errCode.HTTPStatus)
	
	response := ErrorResponse{
		Code:    errCode.Code,
		Message: errCode.Message,
	}
	
	json.NewEncoder(w).Encode(response)
}

// RespondWithErrorDetails writes an error response with additional details
func RespondWithErrorDetails(w http.ResponseWriter, errCode ErrorCode, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errCode.HTTPStatus)
	
	response := ErrorResponse{
		Code:    errCode.Code,
		Message: errCode.Message,
		Details: details,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleServiceError automatically maps service errors to HTTP responses
// This function checks context errors first, then iterates through all error codes
// to find matching ServiceErr mappings, providing automatic error translation
func HandleServiceError(w http.ResponseWriter, err error) {
	// Check context errors first (client disconnection, timeouts)
	if errors.Is(err, context.Canceled) {
		w.WriteHeader(499) // Client Closed Request (nginx convention)
		w.Write([]byte(`{"error":"Request cancelled"}`))
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte(`{"error":"Request timeout"}`))
		return
	}
	
	// Iterate through all error codes to find ServiceErr mapping
	for _, errCode := range AllErrors() {
		if errCode.ServiceErr != nil && errors.Is(err, errCode.ServiceErr) {
			RespondWithError(w, errCode)
			return
		}
	}
	
	// Default: Internal error (don't expose details to client)
	RespondWithError(w, Errors.InternalError)
}

