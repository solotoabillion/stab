package billing

import (
	"context"
	"fmt"
	"net/http"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"

	stripe "github.com/stripe/stripe-go/v76"
	billingportalsession "github.com/stripe/stripe-go/v76/billingportal/session"
)

type CreatePortalSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePortalSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePortalSessionLogic {
	return &CreatePortalSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostCreatePortalSession creates a Stripe Billing Portal session for the user.
func (l *CreatePortalSessionLogic) PostCreatePortalSession(c echo.Context) (resp *types.PortalSessionResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for creating portal session")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Check if user has a Stripe Customer ID
	// User details (including StripeCustomerID) are already available in authedUser
	if authedUser.StripeCustomerID == nil || *authedUser.StripeCustomerID == "" {
		l.Infof("User %s attempted to access billing portal without a Stripe Customer ID", userID.String())
		// Use a more appropriate status code like Bad Request or Forbidden? Using Bad Request.
		return nil, echo.NewHTTPError(http.StatusBadRequest, "No active billing account found. Please subscribe to a plan first.")
	}
	stripeCustomerID := *authedUser.StripeCustomerID

	// 3. Initialize Stripe client
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for CreatePortalSession")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error")
	}

	// 4. Get Return URL from Config
	frontendURL := l.svcCtx.Config.FrontendURL // TODO: Verify FrontendURL exists in config.Config
	if frontendURL == "" {
		l.Error("FRONTEND_URL missing from service config")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Application configuration error")
	}
	// TODO: Make return path configurable?
	returnURL := frontendURL + "/app/billing"

	// 5. Create Stripe Billing Portal Session Params
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(returnURL),
	}

	// 6. Create Stripe Session
	ps, stripeErr := billingportalsession.New(params)
	if stripeErr != nil {
		l.Errorf("Failed to create Stripe billing portal session for user %s (customer %s): %v", userID.String(), stripeCustomerID, stripeErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create billing management session: %s", stripeErr.Error()))
	}

	// 7. Map response
	resp = &types.PortalSessionResponse{
		Success: true,
		Message: "Portal session created successfully",
		URL:     ps.URL,
	}

	l.Infof("Created Stripe billing portal session for user %s (customer %s)", userID.String(), stripeCustomerID)
	return resp, nil
}
