package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	stripe "github.com/stripe/stripe-go/v76"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
)

type CreateCheckoutSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCheckoutSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCheckoutSessionLogic {
	return &CreateCheckoutSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostCreateCheckoutSession creates a Stripe Checkout session for subscribing to a plan.
func (l *CreateCheckoutSessionLogic) PostCreateCheckoutSession(c echo.Context, req *types.CheckoutSessionRequest) (resp *types.CheckoutSessionResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for creating checkout session")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Validate PlanID (basic check, more specific validation might be needed)
	if req.PlanID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "PlanID is required")
	}
	// Assuming PlanID in the request corresponds to the Plan model's string ID.
	// Removed parsing req.PlanID as UUID as it's used directly with FindPlanByID.

	// 3. Fetch User Details (needed for Stripe Customer ID/Email)
	// User details are already available in authedUser from context.

	// 4. Fetch Plan Details from DB
	// 4. Fetch Plan Details using model function
	// Note: PlanID in the request is a UUID string, but the Plan model uses a string ID.
	// Assuming the request PlanID should actually be the string ID used in the Plan model.
	// If req.PlanID is truly meant to be a UUID, the Plan model or this logic needs adjustment.
	// For now, we proceed assuming req.PlanID is the string ID.
	planPtr, dbErr := models.FindPlanByID(l.svcCtx.DB, req.PlanID) // Use req.PlanID directly
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			l.Infof("Checkout session requested for non-existent plan ID: %s", req.PlanID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Selected plan not found")
		}
		l.Errorf("Failed to fetch plan details for ID %s: %v", req.PlanID, dbErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve plan details")
	}
	plan := *planPtr // Dereference

	// 5. Determine Stripe Price ID based on request and plan
	var stripePriceID string
	if req.IsYearly {
		// Check if pointer is nil OR if the dereferenced string is empty
		if plan.StripePriceIDYearly == nil || *plan.StripePriceIDYearly == "" {
			l.Infof("Yearly price requested for plan %s (%s), but no yearly price ID configured", plan.Name, plan.ID) // Changed Warnf, removed .String()
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Yearly pricing is not available for this plan")
		}
		stripePriceID = *plan.StripePriceIDYearly // Dereference pointer
	} else {
		// Use the main StripePriceID field for monthly
		if plan.StripePriceID == "" {
			l.Infof("Monthly price requested for plan %s (%s), but no monthly price ID configured", plan.Name, plan.ID) // Changed Warnf, removed .String()
			// This should ideally not happen if a plan exists, but handle defensively
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Pricing is not available for this plan")
		}
		stripePriceID = plan.StripePriceID
	}

	// 6. Initialize Stripe client
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for CreateCheckoutSession")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error")
	}

	// 7. Get Success/Cancel URLs from Config
	// Assuming FrontendURL is defined in the main config struct
	frontendURL := l.svcCtx.Config.FrontendURL // TODO: Verify FrontendURL exists in config.Config
	if frontendURL == "" {
		l.Error("FRONTEND_URL missing from service config")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Application configuration error")
	}
	// TODO: Make success/cancel paths configurable?
	successURL := frontendURL + "/app/billing?session_id={CHECKOUT_SESSION_ID}&status=success"
	cancelURL := frontendURL + "/app/billing?status=cancel"

	// 8. Create Stripe Checkout Session Params
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems:         []*stripe.CheckoutSessionLineItemParams{{Price: stripe.String(stripePriceID), Quantity: stripe.Int64(1)}},
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(userID.String()), // Pass user UUID string
		// Allow promotion codes if needed
		// AllowPromotionCodes: stripe.Bool(true),
		// Pre-fill email and potentially customer ID
		CustomerEmail: stripe.String(authedUser.Email),
	}
	if authedUser.StripeCustomerID != nil && *authedUser.StripeCustomerID != "" {
		params.Customer = authedUser.StripeCustomerID
		// If customer exists, don't send email again unless needed
		params.CustomerEmail = nil
	}

	// 9. Create Stripe Session
	s, stripeErr := checkoutsession.New(params)
	if stripeErr != nil {
		l.Errorf("Failed to create Stripe checkout session for user %s, plan %s (%s): %v", userID.String(), plan.Name, stripePriceID, stripeErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to initiate billing session: %s", stripeErr.Error()))
	}

	// 10. Map response
	resp = &types.CheckoutSessionResponse{
		Success:   true,
		Message:   "Checkout session created successfully",
		SessionID: s.ID,
		URL:       s.URL,
	}

	l.Infof("Created Stripe checkout session %s for user %s, plan %s (%s)", s.ID, userID.String(), plan.Name, stripePriceID)
	return resp, nil
}
