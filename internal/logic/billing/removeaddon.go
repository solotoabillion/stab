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

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscriptionitem"
)

type RemoveAddonLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveAddonLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveAddonLogic {
	return &RemoveAddonLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveAddonLogic) DeleteRemoveAddon(c echo.Context, req *types.AddonRequest) (resp *types.AddonResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for removing addon")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Parse the addon ID from request
	resourceUUID, err := uuid.Parse(req.AddonID)
	if err != nil {
		l.Errorf("Invalid AddonID format in request: %s, error: %v", req.AddonID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid addon ID format")
	}

	// 3. Initialize Stripe client
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for RemoveAddon")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error")
	}

	// 4. Find the Local SubscriptionItem Record
	currentSubscriptionPtr, dbErr := models.FindLatestActiveSubscriptionByUserID(l.svcCtx.DB, userID)
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "No active subscription found")
		}
		l.Errorf("Error fetching active subscription for user %s: %v", userID, dbErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve subscription details")
	}
	currentSubscription := *currentSubscriptionPtr

	// 5. Find the subscription item to remove
	itemToRemove, err := models.FindSubscriptionItemByTypeAndResource(l.svcCtx.DB, currentSubscription.ID, "reserved_domain", resourceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Try with custom_domain type if not found as reserved_domain
			itemToRemove, err = models.FindSubscriptionItemByTypeAndResource(l.svcCtx.DB, currentSubscription.ID, "custom_domain", resourceUUID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					l.Infof("Add-on not found for removal: User %s, Resource %s", userID, req.AddonID)
					return nil, echo.NewHTTPError(http.StatusNotFound, "Add-on not found for this resource")
				}
				l.Errorf("Error finding subscription item to remove: %v", err)
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error finding add-on")
			}
		} else {
			l.Errorf("Error finding subscription item to remove: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error finding add-on")
		}
	}

	// 6. Delete Item from Stripe Subscription
	_, err = subscriptionitem.Del(itemToRemove.StripeSubscriptionItemID, nil)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			l.Infof("Stripe subscription item %s already deleted, proceeding to delete local record.", itemToRemove.StripeSubscriptionItemID)
		} else {
			l.Errorf("Stripe API error removing subscription item %s: %v", itemToRemove.StripeSubscriptionItemID, err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update billing: %s", err.Error()))
		}
	}

	// 7. Delete Local SubscriptionItem Record
	if err := l.svcCtx.DB.Delete(itemToRemove).Error; err != nil {
		l.Errorf("Failed to delete local subscription item record %s after removing from Stripe: %v", itemToRemove.ID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update local add-on status")
	}

	l.Infof("Successfully removed add-on: User %s, Resource %s, Stripe Item %s", userID, req.AddonID, itemToRemove.StripeSubscriptionItemID)

	// 8. Return success response
	resp = &types.AddonResponse{
		Success: true,
		Message: "Add-on removed successfully",
		AddonID: req.AddonID,
	}
	return resp, nil
}
