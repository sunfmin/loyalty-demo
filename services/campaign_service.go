package services

import (
	"context"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
)

// CampaignService defines the interface for promotional campaign operations
// All methods accept context.Context as first parameter per constitutional requirement
type CampaignService interface {
	// CreateCampaign creates a new promotional campaign
	// Returns error if dates invalid or campaign already exists
	CreateCampaign(ctx context.Context, req *loyaltyv1.CreateCampaignRequest) (*loyaltyv1.PromotionalCampaign, error)

	// ListCampaigns returns all campaigns with optional filtering
	// No authentication required for viewing active campaigns
	ListCampaigns(ctx context.Context, req *loyaltyv1.ListCampaignsRequest) (*loyaltyv1.ListCampaignsResponse, error)

	// GetActiveCampaigns returns currently active campaigns (used internally for point calculation)
	GetActiveCampaigns(ctx context.Context) ([]*loyaltyv1.PromotionalCampaign, error)
}

// Stub implementation for now (TODO: implement in campaign_service_impl.go)
type campaignService struct {
	db interface{}
}

// NewCampaignService creates a new campaign service
func NewCampaignService(db interface{}) CampaignService {
	return &campaignService{db: db}
}

func (s *campaignService) CreateCampaign(ctx context.Context, req *loyaltyv1.CreateCampaignRequest) (*loyaltyv1.PromotionalCampaign, error) {
	// Stub implementation
	return &loyaltyv1.PromotionalCampaign{
		Name: req.Name,
	}, nil
}

func (s *campaignService) ListCampaigns(ctx context.Context, req *loyaltyv1.ListCampaignsRequest) (*loyaltyv1.ListCampaignsResponse, error) {
	// Stub implementation
	return &loyaltyv1.ListCampaignsResponse{
		Campaigns: []*loyaltyv1.PromotionalCampaign{},
	}, nil
}

func (s *campaignService) GetActiveCampaigns(ctx context.Context) ([]*loyaltyv1.PromotionalCampaign, error) {
	// Stub implementation
	return []*loyaltyv1.PromotionalCampaign{}, nil
}

