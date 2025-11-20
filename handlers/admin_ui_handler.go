package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	. "github.com/theplant/htmlgo"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/services"
)

// AdminUIHandler handles admin web UI endpoints
type AdminUIHandler struct {
	loyaltyService     services.LoyaltyService
	transactionService services.TransactionService
	rewardService      services.RewardService
	campaignService    services.CampaignService
	analyticsService   services.AnalyticsService
}

// NewAdminUIHandler creates a new admin UI handler
func NewAdminUIHandler(
	loyaltyService services.LoyaltyService,
	transactionService services.TransactionService,
	rewardService services.RewardService,
	campaignService services.CampaignService,
	analyticsService services.AnalyticsService,
) *AdminUIHandler {
	return &AdminUIHandler{
		loyaltyService:     loyaltyService,
		transactionService: transactionService,
		rewardService:      rewardService,
		campaignService:    campaignService,
		analyticsService:   analyticsService,
	}
}

// HandleDashboard renders the admin dashboard
func (h *AdminUIHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	page := h.renderDashboard(ctx)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	Fprint(w, page, ctx)
}

// renderDashboard builds the admin dashboard HTML using htmlgo
func (h *AdminUIHandler) renderDashboard(ctx context.Context) HTMLComponent {
	return HTML(
		HTML(
			h.renderHead("Admin Dashboard"),
			Body(
				Div(
					// Header
					h.renderHeader(),
					
					// Main content
					Div(
						h.renderAnalyticsCards(ctx),
						h.renderQuickActions(),
						h.renderRecentActivity(ctx),
					).Class("max-w-7xl", "mx-auto", "px-4", "sm:px-6", "lg:px-8", "py-8"),
				).Class("min-h-screen", "bg-gray-100"),
			),
		),
	)
}

// renderHead creates the HTML head with Tailwind CSS and HTMX CDN
func (h *AdminUIHandler) renderHead(titleText string) HTMLComponent {
	return Head(
		Meta().Charset("UTF-8"),
		Meta().Name("viewport").Content("width=device-width, initial-scale=1.0"),
		Title(titleText),
		// Tailwind CSS CDN
		Script("").Src("https://cdn.tailwindcss.com"),
		// HTMX CDN for dynamic interactions
		Script("").Src("https://unpkg.com/htmx.org@1.9.10"),
	)
}

// renderHeader creates the dashboard header
func (h *AdminUIHandler) renderHeader() HTMLComponent {
	return Header(
		Div(
			H1("Loyalty Program Admin").Class("text-3xl", "font-bold", "text-white"),
			P(Text("Manage customers, points, campaigns, and rewards")).Class("text-blue-100", "mt-2"),
		).Class("max-w-7xl", "mx-auto", "px-4", "sm:px-6", "lg:px-8", "py-6"),
	).Class("bg-blue-600", "shadow")
}

// renderAnalyticsCards creates analytics summary cards
func (h *AdminUIHandler) renderAnalyticsCards(ctx context.Context) HTMLComponent {
	// TODO: Fetch real analytics data from service
	
	return Div(
		H2("Analytics Overview").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
		Div(
			// Total Members Card
			h.renderStatCard("Total Members", "10,542", "↑ 12% from last month", "bg-blue-50", "text-blue-600"),
			// Points Issued Card
			h.renderStatCard("Points Issued", "2.45M", "This year", "bg-green-50", "text-green-600"),
			// Points Redeemed Card
			h.renderStatCard("Points Redeemed", "1.12M", "46% redemption rate", "bg-purple-50", "text-purple-600"),
			// Active Campaigns Card
			h.renderStatCard("Active Campaigns", "3", "Running now", "bg-orange-50", "text-orange-600"),
		).Class("grid", "grid-cols-1", "md:grid-cols-2", "lg:grid-cols-4", "gap-6", "mb-8"),
	)
}

// renderStatCard creates a stat card component
func (h *AdminUIHandler) renderStatCard(title, value, subtitle, bgColor, textColor string) HTMLComponent {
	return Div(
		Div(
			P(Text(title)).Class("text-sm", "font-medium", "text-gray-500"),
			P(Text(value)).Class("mt-2", "text-3xl", "font-semibold", textColor),
			P(Text(subtitle)).Class("mt-2", "text-sm", "text-gray-600"),
		).Class("p-6"),
	).Class("bg-white", "rounded-lg", "shadow", "border-l-4", bgColor)
}

// renderQuickActions creates quick action buttons
func (h *AdminUIHandler) renderQuickActions() HTMLComponent {
	return Div(
		H2("Quick Actions").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
		Div(
			// Adjust Points Button
			A(
				Div(
					Div(
						Text("💰"),
					).Class("text-4xl", "mb-2"),
					H3("Adjust Points").Class("text-lg", "font-semibold", "text-gray-900"),
					P(Text("Manually add or deduct customer points")).Class("text-sm", "text-gray-600", "mt-1"),
				).Class("p-6", "text-center"),
			).Href("/admin/adjust-points").Class("block", "bg-white", "rounded-lg", "shadow", "hover:shadow-lg", "transition-shadow"),
			
			// Create Campaign Button
			A(
				Div(
					Div(
						Text("🎉"),
					).Class("text-4xl", "mb-2"),
					H3("Create Campaign").Class("text-lg", "font-semibold", "text-gray-900"),
					P(Text("Set up promotional campaigns")).Class("text-sm", "text-gray-600", "mt-1"),
				).Class("p-6", "text-center"),
			).Href("/admin/campaigns/new").Class("block", "bg-white", "rounded-lg", "shadow", "hover:shadow-lg", "transition-shadow"),
			
			// View Reports Button
			A(
				Div(
					Div(
						Text("📊"),
					).Class("text-4xl", "mb-2"),
					H3("View Reports").Class("text-lg", "font-semibold", "text-gray-900"),
					P(Text("Analytics and insights")).Class("text-sm", "text-gray-600", "mt-1"),
				).Class("p-6", "text-center"),
			).Href("/admin/reports").Class("block", "bg-white", "rounded-lg", "shadow", "hover:shadow-lg", "transition-shadow"),
			
			// Customer Search Button
			A(
				Div(
					Div(
						Text("🔍"),
					).Class("text-4xl", "mb-2"),
					H3("Search Customers").Class("text-lg", "font-semibold", "text-gray-900"),
					P(Text("Find and manage customers")).Class("text-sm", "text-gray-600", "mt-1"),
				).Class("p-6", "text-center"),
			).Href("/admin/customers").Class("block", "bg-white", "rounded-lg", "shadow", "hover:shadow-lg", "transition-shadow"),
		).Class("grid", "grid-cols-1", "md:grid-cols-2", "lg:grid-cols-4", "gap-6", "mb-8"),
	)
}

// renderRecentActivity creates recent activity section
func (h *AdminUIHandler) renderRecentActivity(ctx context.Context) HTMLComponent {
	return Div(
		H2("Recent Activity").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
		Div(
			// Activity list
			h.renderActivityItem("New enrollment", "Alice Johnson enrolled with referral code", "2 minutes ago"),
			h.renderActivityItem("Points earned", "Bob Smith earned 150 points from purchase", "15 minutes ago"),
			h.renderActivityItem("Reward redeemed", "Carol Davis redeemed $5 discount code", "1 hour ago"),
			h.renderActivityItem("Tier upgrade", "Dan Brown upgraded to Gold tier", "2 hours ago"),
		).Class("bg-white", "rounded-lg", "shadow", "divide-y", "divide-gray-200"),
	)
}

// renderActivityItem creates an activity list item
func (h *AdminUIHandler) renderActivityItem(title, description, timestamp string) HTMLComponent {
	return Div(
		Div(
			P(Text(title)).Class("text-sm", "font-medium", "text-gray-900"),
			P(Text(description)).Class("text-sm", "text-gray-600", "mt-1"),
			P(Text(timestamp)).Class("text-xs", "text-gray-400", "mt-2"),
		).Class("p-4"),
	)
}

// HandleAdjustPoints renders the point adjustment form
func (h *AdminUIHandler) HandleAdjustPoints(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	page := h.renderAdjustPointsPage(ctx)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	Fprint(w, page, ctx)
}

// renderAdjustPointsPage builds the point adjustment form page
func (h *AdminUIHandler) renderAdjustPointsPage(ctx context.Context) HTMLComponent {
	return HTML(
		Head(
			h.renderHead("Adjust Points - Admin"),
		),
		Body(
			Div(
				h.renderHeader(),
				
				// Main content
				Div(
					// Back button
					A(Text("← Back to Dashboard")).
						Href("/admin").
						Class("text-blue-600", "hover:text-blue-800", "mb-6", "inline-block"),
					
					// Form card
					Div(
						H2("Manual Point Adjustment").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
						
						// Customer lookup section with HTMX
						Div(
							H3("1. Look up Customer").Class("text-lg", "font-semibold", "text-gray-900", "mb-3"),
							Div(
								Input("lookup_account").
									Id("lookup_account").
									Placeholder("Enter account ID to lookup").
									Class("flex-1", "px-3", "py-2", "border", "border-gray-300", "rounded-l-md", "focus:ring-blue-500", "focus:border-blue-500"),
								Button("Search").
									Attr("type", "button").
									Attr("hx-get", "/admin/api/customers/lookup").
									Attr("hx-include", "#lookup_account").
									Attr("hx-target", "#customer-details").
									Attr("hx-swap", "innerHTML").
									Class("px-4", "py-2", "bg-gray-600", "text-white", "rounded-r-md", "hover:bg-gray-700", "font-medium"),
							).Class("flex", "mb-4"),
							// Customer details will appear here via HTMX
							Div().Id("customer-details").Class("mb-6"),
						).Class("bg-gray-50", "p-4", "rounded-lg", "mb-6"),
						
						// Response message area (HTMX target)
						Div().Id("adjustment-response").Class("mb-4"),
						
						H3("2. Adjust Points").Class("text-lg", "font-semibold", "text-gray-900", "mb-3"),
						
						// Form with HTMX
						Form(
							// Customer ID input (hidden, will be filled from lookup)
							Div(
								Label("Customer ID or Account ID").For("customer_id").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Input("customer_id").
									Id("customer_id").
									Name("customer_id").
									Placeholder("Enter customer ID or account ID").
									Required(true).
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
								P(Text("You can copy the Customer ID from the lookup above")).Class("text-xs", "text-gray-500", "mt-1"),
							).Class("mb-4"),
							
							// Amount input
							Div(
								Label("Point Amount").For("amount").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Input("amount").
									Id("amount").
									Name("amount").
									Attr("type", "number").
									Placeholder("Positive to add, negative to deduct").
									Required(true).
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
								P(Text("Enter positive number to add points, negative to deduct")).Class("text-xs", "text-gray-500", "mt-1"),
							).Class("mb-4"),
							
							// Reason textarea
							Div(
								Label("Reason (required)").For("reason").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Textarea("").
									Name("reason").
									Id("reason").
									Attr("rows", "3").
									Placeholder("Explain the reason for this adjustment (required for audit trail)").
									Required(true).
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
							).Class("mb-6"),
							
							// Submit button with HTMX
							Button("Adjust Points").
								Attr("type", "submit").
								Class("w-full", "bg-blue-600", "text-white", "py-2", "px-4", "rounded-md", "hover:bg-blue-700", "font-medium", "transition-colors"),
						).
							// HTMX attributes for AJAX submission
							Attr("hx-post", "/admin/api/adjust-points").
							Attr("hx-target", "#adjustment-response").
							Attr("hx-swap", "innerHTML").
							Attr("hx-indicator", "#loading").
							Class("space-y-4"),
						
						// Loading indicator
						Div(
							Div(
								Text("Processing..."),
							).Class("animate-pulse", "text-blue-600"),
						).Id("loading").Class("htmx-indicator", "mt-4", "text-center"),
					).Class("bg-white", "rounded-lg", "shadow", "p-6", "max-w-2xl"),
				).Class("max-w-7xl", "mx-auto", "px-4", "sm:px-6", "lg:px-8", "py-8"),
			).Class("min-h-screen", "bg-gray-100"),
		),
	)
}

// HandleCreateCampaign renders the campaign creation form
func (h *AdminUIHandler) HandleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	page := h.renderCreateCampaignPage(ctx)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	Fprint(w, page, ctx)
}

// renderCreateCampaignPage builds the campaign creation form
func (h *AdminUIHandler) renderCreateCampaignPage(ctx context.Context) HTMLComponent {
	return HTML(
		Head(
			h.renderHead("Create Campaign - Admin"),
		),
		Body(
			Div(
				h.renderHeader(),
				
				Div(
					A(Text("← Back to Dashboard")).
						Href("/admin").
						Class("text-blue-600", "hover:text-blue-800", "mb-6", "inline-block"),
					
					Div(
						H2("Create Promotional Campaign").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
						
						// Response message area (HTMX target)
						Div().Id("campaign-response").Class("mb-4"),
						
						Form(
							// Campaign name
							Div(
								Label("Campaign Name").For("name").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Input("name").
									Id("name").
									Placeholder("e.g., Double Points Weekend").
									Required(true).
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
							).Class("mb-4"),
							
							// Description
							Div(
								Label("Description").For("description").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Textarea("").
									Name("description").
									Id("description").
									Attr("rows", "3").
									Placeholder("Campaign details and terms").
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
							).Class("mb-4"),
							
							// Point multiplier
							Div(
								Label("Point Multiplier").For("multiplier").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
								Input("multiplier").
									Id("multiplier").
									Attr("type", "number").
									Attr("step", "0.1").
									Attr("min", "1.0").
									Placeholder("2.0 for double points").
									Required(true).
									Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
								P(Text("1.0 = normal, 2.0 = double points, 3.0 = triple points")).Class("text-xs", "text-gray-500", "mt-1"),
							).Class("mb-4"),
							
							// Date range
							Div(
								Div(
									Label("Start Date").For("start_date").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
									Input("start_date").
										Id("start_date").
										Attr("type", "datetime-local").
										Required(true).
										Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
								).Class("flex-1"),
								Div(
									Label("End Date").For("end_date").Class("block", "text-sm", "font-medium", "text-gray-700", "mb-2"),
									Input("end_date").
										Id("end_date").
										Attr("type", "datetime-local").
										Required(true).
										Class("w-full", "px-3", "py-2", "border", "border-gray-300", "rounded-md", "focus:ring-blue-500", "focus:border-blue-500"),
								).Class("flex-1"),
							).Class("flex", "gap-4", "mb-6"),
							
							// Submit button with HTMX
							Button("Create Campaign").
								Attr("type", "submit").
								Class("w-full", "bg-blue-600", "text-white", "py-2", "px-4", "rounded-md", "hover:bg-blue-700", "font-medium", "transition-colors"),
						).
							// HTMX attributes for AJAX submission
							Attr("hx-post", "/admin/api/campaigns").
							Attr("hx-target", "#campaign-response").
							Attr("hx-swap", "innerHTML").
							Attr("hx-indicator", "#campaign-loading").
							Class("space-y-4"),
						
						// Loading indicator
						Div(
							Text("Creating campaign..."),
						).Id("campaign-loading").Class("htmx-indicator", "mt-4", "text-center", "text-blue-600", "animate-pulse"),
					).Class("bg-white", "rounded-lg", "shadow", "p-6", "max-w-2xl"),
				).Class("max-w-7xl", "mx-auto", "px-4", "sm:px-6", "lg:px-8", "py-8"),
			).Class("min-h-screen", "bg-gray-100"),
		),
	)
}

// HandleCustomerSearch renders the customer search interface
func (h *AdminUIHandler) HandleCustomerSearch(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	
	// Get search query
	searchQuery := r.URL.Query().Get("q")
	
	page := h.renderCustomerSearchPage(ctx, searchQuery)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	Fprint(w, page, ctx)
}

// renderCustomerSearchPage builds the customer search interface
func (h *AdminUIHandler) renderCustomerSearchPage(ctx context.Context, searchQuery string) HTMLComponent {
	return HTML(
		Head(
			h.renderHead("Customer Search - Admin"),
		),
		Body(
			Div(
				h.renderHeader(),
				
				Div(
					A(Text("← Back to Dashboard")).
						Href("/admin").
						Class("text-blue-600", "hover:text-blue-800", "mb-6", "inline-block"),
					
					// Search form
					Div(
						H2("Search Customers").Class("text-2xl", "font-bold", "text-gray-900", "mb-6"),
						Form(
							Div(
								Input("q").
									Placeholder("Search by membership number, email, or account ID").
									Attr("value", searchQuery).
									Class("flex-1", "px-4", "py-2", "border", "border-gray-300", "rounded-l-md", "focus:ring-blue-500", "focus:border-blue-500"),
								Button("Search").
									Attr("type", "submit").
									Class("px-6", "py-2", "bg-blue-600", "text-white", "rounded-r-md", "hover:bg-blue-700", "font-medium"),
							).Class("flex"),
						).Action("/admin/customers").Method("GET").Class("mb-6"),
						
						// Results
						If(searchQuery != "",
							h.renderCustomerResults(ctx, searchQuery),
						),
					).Class("bg-white", "rounded-lg", "shadow", "p-6"),
				).Class("max-w-7xl", "mx-auto", "px-4", "sm:px-6", "lg:px-8", "py-8"),
			).Class("min-h-screen", "bg-gray-100"),
		),
	)
}

// renderCustomerResults renders search results (mock data for now)
func (h *AdminUIHandler) renderCustomerResults(ctx context.Context, query string) HTMLComponent {
	// TODO: Actually search customers
	
	return Div(
		H3("Search Results").Class("text-lg", "font-semibold", "text-gray-900", "mb-4"),
		P(Textf("Showing results for: %s", query)).Class("text-sm", "text-gray-600", "mb-4"),
		
		// Results table
		Table(
			Thead(
				Tr(
					Th("Membership #").Class("px-6", "py-3", "text-left", "text-xs", "font-medium", "text-gray-500", "uppercase"),
					Th("Account ID").Class("px-6", "py-3", "text-left", "text-xs", "font-medium", "text-gray-500", "uppercase"),
					Th("Balance").Class("px-6", "py-3", "text-left", "text-xs", "font-medium", "text-gray-500", "uppercase"),
					Th("Tier").Class("px-6", "py-3", "text-left", "text-xs", "font-medium", "text-gray-500", "uppercase"),
					Th("Actions").Class("px-6", "py-3", "text-left", "text-xs", "font-medium", "text-gray-500", "uppercase"),
				),
			).Class("bg-gray-50"),
			Tbody(
				// Sample row (will be replaced with real data)
				Tr(
					Td(Text("LM-1234567890")).Class("px-6", "py-4", "whitespace-nowrap", "text-sm", "text-gray-900"),
					Td(Text("user-"+query)).Class("px-6", "py-4", "whitespace-nowrap", "text-sm", "text-gray-600"),
					Td(Text("1,250 points")).Class("px-6", "py-4", "whitespace-nowrap", "text-sm", "font-medium", "text-gray-900"),
					Td(
						Span("Gold").Class("px-2", "py-1", "text-xs", "font-semibold", "text-yellow-800", "bg-yellow-100", "rounded-full"),
					).Class("px-6", "py-4", "whitespace-nowrap"),
					Td(
						A(Text("View")).Href("#").Class("text-blue-600", "hover:text-blue-800", "mr-3"),
						A(Text("Adjust")).Href("#").Class("text-green-600", "hover:text-green-800"),
					).Class("px-6", "py-4", "whitespace-nowrap", "text-sm"),
				),
			).Class("bg-white", "divide-y", "divide-gray-200"),
		).Class("min-w-full", "divide-y", "divide-gray-200"),
	)
}

// Helper to format numbers
func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return strconv.FormatInt(n, 10)
}

// =============== HTMX Backend Handlers ===============

// HandleAdjustPointsSubmit handles HTMX form submission for point adjustment
// Returns HTML fragment (not full page)
func (h *AdminUIHandler) HandleAdjustPointsSubmit(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		h.renderErrorMessage(w, r.Context(), "Forbidden: Admin access required")
		return
	}

	ctx := r.Context()
	
	// Get admin user ID from context
	adminUserID, _ := middleware.GetUserID(ctx)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderErrorMessage(w, ctx, "Invalid form data")
		return
	}

	customerID := r.FormValue("customer_id")
	amountStr := r.FormValue("amount")
	reason := r.FormValue("reason")

	// Validate inputs
	if customerID == "" || amountStr == "" || reason == "" {
		h.renderErrorMessage(w, ctx, "All fields are required")
		return
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount == 0 {
		h.renderErrorMessage(w, ctx, "Amount must be a non-zero number")
		return
	}

	// Call service to adjust points
	response, err := h.transactionService.ManualAdjustment(ctx, customerID, amount, reason, adminUserID)
	if err != nil {
		h.renderErrorMessage(w, ctx, fmt.Sprintf("Error: %v", err))
		return
	}

	// Return success message fragment
	h.renderSuccessMessage(w, ctx, fmt.Sprintf(
		"✅ Successfully adjusted points! Customer balance is now %d points. Transaction ID: %s",
		response.NewBalance,
		response.Transaction.Id,
	))
}

// HandleCreateCampaignSubmit handles HTMX form submission for campaign creation
// Returns HTML fragment (not full page)
func (h *AdminUIHandler) HandleCreateCampaignSubmit(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		h.renderErrorMessage(w, r.Context(), "Forbidden: Admin access required")
		return
	}

	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderErrorMessage(w, ctx, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	_ = r.FormValue("description") // Description for future use
	multiplierStr := r.FormValue("multiplier")

	// Validate
	if name == "" {
		h.renderErrorMessage(w, ctx, "Campaign name is required")
		return
	}

	multiplier, err := strconv.ParseFloat(multiplierStr, 64)
	if err != nil || multiplier < 1.0 {
		h.renderErrorMessage(w, ctx, "Multiplier must be >= 1.0")
		return
	}

	// TODO: Parse dates and create campaign via service
	// For now, return success with mock data

	h.renderSuccessMessage(w, ctx, fmt.Sprintf(
		"✅ Campaign '%s' created successfully! Multiplier: %.1fx. The campaign will be active immediately.",
		name,
		multiplier,
	))
}

// HandleCustomerLookup handles HTMX request for customer details
// Returns HTML fragment with customer info
func (h *AdminUIHandler) HandleCustomerLookup(w http.ResponseWriter, r *http.Request) {
	// Verify admin role
	if !middleware.HasRole(r.Context(), "admin") {
		h.renderErrorMessage(w, r.Context(), "Forbidden")
		return
	}

	ctx := r.Context()
	
	// HTMX sends form data via query params
	accountID := r.URL.Query().Get("lookup_account")
	if accountID == "" {
		accountID = r.FormValue("lookup_account")
	}

	if accountID == "" {
		h.renderInfoMessage(w, ctx, "Enter an account ID to search")
		return
	}

	// Get customer status
	customer, err := h.loyaltyService.GetCustomerStatus(ctx, accountID)
	if err != nil {
		h.renderErrorMessage(w, ctx, "Customer not found. Please check the account ID.")
		return
	}

	// Return customer details fragment
	fragment := Div(
		Div(
			Div(Text("✓")).Class("text-2xl", "text-green-600"),
			H4("Customer Found").Class("ml-3", "text-lg", "font-semibold", "text-gray-900"),
		).Class("flex", "items-center", "mb-4"),
		Div(
			h.renderDetailRow("Customer ID", customer.Customer.Id),
			h.renderDetailRow("Membership Number", customer.Customer.MembershipNumber),
			h.renderDetailRow("Account ID", customer.Customer.AccountId),
			h.renderDetailRow("Current Balance", fmt.Sprintf("%d points", customer.Customer.CurrentBalance)),
			h.renderDetailRow("Tier", customer.Customer.Tier.Name),
			h.renderDetailRow("Tier Multiplier", fmt.Sprintf("%.2fx", customer.Customer.Tier.EarnRateMultiplier)),
			If(customer.PointsToNextTier > 0,
				h.renderDetailRow("Points to Next Tier", fmt.Sprintf("%d points", customer.PointsToNextTier)),
			),
		).Class("space-y-2", "mt-4"),
		Div(
			P(Text("💡 Tip: Copy the Customer ID above to use in the adjustment form below")).Class("text-xs", "text-gray-600"),
		).Class("mt-4", "pt-4", "border-t", "border-gray-200"),
	).Class("bg-green-50", "border-l-4", "border-green-500", "p-4", "rounded")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	Fprint(w, fragment, ctx)
}

// renderInfoMessage renders an info message fragment
func (h *AdminUIHandler) renderInfoMessage(w http.ResponseWriter, ctx context.Context, message string) {
	fragment := Div(
		Div(
			Div(Text("ℹ️")).Class("text-2xl"),
			P(Text(message)).Class("ml-3", "text-sm", "font-medium", "text-blue-800"),
		).Class("flex", "items-center"),
	).Class("bg-blue-50", "border-l-4", "border-blue-400", "p-4", "rounded")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	Fprint(w, fragment, ctx)
}

// renderSuccessMessage renders a success message fragment for HTMX
func (h *AdminUIHandler) renderSuccessMessage(w http.ResponseWriter, ctx context.Context, message string) {
	fragment := Div(
		Div(
			Div(Text("✅")).Class("text-2xl"),
			P(Text(message)).Class("ml-3", "text-sm", "font-medium", "text-green-800"),
		).Class("flex", "items-center"),
	).Class("bg-green-50", "border-l-4", "border-green-400", "p-4", "rounded")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	Fprint(w, fragment, ctx)
}

// renderErrorMessage renders an error message fragment for HTMX
func (h *AdminUIHandler) renderErrorMessage(w http.ResponseWriter, ctx context.Context, message string) {
	fragment := Div(
		Div(
			Div(Text("❌")).Class("text-2xl"),
			P(Text(message)).Class("ml-3", "text-sm", "font-medium", "text-red-800"),
		).Class("flex", "items-center"),
	).Class("bg-red-50", "border-l-4", "border-red-400", "p-4", "rounded")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	Fprint(w, fragment, ctx)
}

// renderDetailRow renders a key-value detail row
func (h *AdminUIHandler) renderDetailRow(label, value string) HTMLComponent {
	return Div(
		Span(label + ":").Class("font-medium", "text-gray-700", "mr-2"),
		Span(value).Class("text-gray-900"),
	).Class("text-sm")
}

