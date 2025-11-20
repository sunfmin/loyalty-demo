package services

import "errors"

// Domain-specific sentinel errors for type-safe error checking
// These errors are used internally by services and wrapped with context using fmt.Errorf("%w", err)
// Use errors.Is() to check for these errors in calling code

var (
	// Enrollment errors
	ErrNotEnrolled        = errors.New("customer not enrolled in loyalty program")
	ErrAlreadyEnrolled    = errors.New("customer already enrolled")
	ErrInvalidReferralCode = errors.New("invalid referral code")
	
	// Transaction errors
	ErrInsufficientBalance = errors.New("insufficient point balance")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrMissingRequired     = errors.New("required field missing")
	ErrInvalidType         = errors.New("invalid type")
	ErrValueOutOfRange     = errors.New("value out of range")
	
	// Reward errors
	ErrRewardNotFound  = errors.New("reward not found")
	ErrRewardInactive  = errors.New("reward is not active")
	
	// Redemption errors
	ErrRedemptionNotFound         = errors.New("redemption not found")
	ErrRedemptionAlreadyReversed  = errors.New("redemption already reversed")
	
	// Campaign errors
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrInvalidDateRange = errors.New("invalid date range")
	
	// General errors
	ErrInvalidRequest = errors.New("invalid request")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrTemplateNotFound = errors.New("template not found")
)

