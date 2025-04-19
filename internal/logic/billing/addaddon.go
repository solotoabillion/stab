package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscriptionitem"
)

type AddAddonLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddAddonLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddAddonLogic {
	return &AddAddonLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostAddAddon adds a billable item to the user's subscription.
// TODO: Verify the request type. The placeholder uses 'types.AddonRequest' (path param only),
// but adding requires ItemType and ResourceID in the body. Assume 'req' contains these fields
// (e.g., from a hypothetical 'types.AddAddonRequestBody' defined in the .api file).
func (l *AddAddonLogic) PostAddAddon(c echo.Context, req *types.AddAddonRequestBody) (resp *types.AddonResponse, err error) { // Assuming AddAddonRequestBody exists
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for adding addon")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Validate request body fields (assuming they exist on req)
	// Go-zero validation should handle basic checks based on tags in types.go
	if req.ItemType == "" || req.ResourceID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "ItemType and ResourceID are required")
	}
	resourceUUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		l.Errorf("Invalid ResourceID format in request: %s, error: %v", req.ResourceID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid resource ID format")
	}

	// 3. Get Stripe Price ID from Config
	var stripePriceID string
	switch req.ItemType {
	case "reserved_domain":
		stripePriceID = l.svcCtx.Config.Stripe.PriceIDs.ReservedDomain // Assuming config structure
	case "custom_domain":
		stripePriceID = l.svcCtx.Config.Stripe.PriceIDs.CustomDomain // Assuming config structure
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid item_type specified")
	}
	if stripePriceID == "" {
		l.Errorf("Stripe Price ID for item_type '%s' is not configured in service config", req.ItemType)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error for add-on type")
	}

	// 4. Initialize Stripe client
	// TODO: Consider initializing Stripe client once in svcCtx if used frequently
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for AddAddon")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error")
	}

	// --- Start DB Transaction ---
	// --- Refactored Logic (Stripe call outside transaction) ---

	// 5. Fetch User's Active Subscription using model function
	currentSubscriptionPtr, dbErr := models.FindLatestActiveSubscriptionByUserID(l.svcCtx.DB, userID)
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "No active subscription found to add item to")
		}
		l.Errorf("Error fetching active subscription for user %s: %v", userID, dbErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve subscription details")
	}
	currentSubscription := *currentSubscriptionPtr

	// 6. Check if Add-on Already Exists in DB using model function
	_, checkErr := models.FindSubscriptionItemByTypeAndResource(l.svcCtx.DB, currentSubscription.ID, req.ItemType, resourceUUID)
	if checkErr == nil { // Item found
		l.Infof("Attempted to add duplicate add-on: User %s, Type %s, Resource %s", userID, req.ItemType, req.ResourceID)
		return nil, echo.NewHTTPError(http.StatusConflict, "This add-on already exists for your subscription")
	} else if !errors.Is(checkErr, gorm.ErrRecordNotFound) { // Unexpected DB error
		l.Errorf("Error checking for existing add-on: %v", checkErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error checking add-on")
	}
	// If ErrRecordNotFound, proceed.

	// 7. Add Item to Stripe Subscription (External Call)
	params := &stripe.SubscriptionItemParams{
		Subscription: stripe.String(currentSubscription.StripeSubscriptionID),
		Price:        stripe.String(stripePriceID),
		Quantity:     stripe.Int64(1),
	}
	newItem, stripeErr := subscriptionitem.New(params)
	if stripeErr != nil {
		l.Errorf("Stripe API error adding subscription item: %v", stripeErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update billing: %s", stripeErr.Error()))
	}

	// 8. Create Local SubscriptionItem Record using model function
	dbItem := models.SubscriptionItem{
		SubscriptionID:           currentSubscription.ID,
		StripeSubscriptionItemID: newItem.ID,
		StripePriceID:            stripePriceID,
		ItemType:                 req.ItemType,
		RelatedResourceID:        &resourceUUID, // Pass pointer
		Quantity:                 1,
	}
	// Use a separate DB call, not within the previous transaction scope
	if createErr := models.CreateSubscriptionItem(l.svcCtx.DB, &dbItem); createErr != nil {
		l.Errorf("CRITICAL: Failed to create local subscription item record after adding to Stripe (Stripe Item ID: %s): %v", newItem.ID, createErr)
		// TODO: Implement reconciliation logic (e.g., queue a job, alert admin)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to save add-on details locally after billing update")
	}

	l.Infof("Successfully added add-on: User %s, Type %s, Resource %s, Stripe Item %s, Local Item %s", userID, req.ItemType, req.ResourceID, newItem.ID, dbItem.ID.String())

	// 11. Return success response
	resp = &types.AddonResponse{
		Success: true,
		Message: "Add-on added successfully",
		AddonID: req.ResourceID, // Return the resource ID as the identifier for the addon
	}
	return resp, nil
}
