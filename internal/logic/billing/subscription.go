package billing

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SubscriptionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubscriptionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubscriptionLogic {
	return &SubscriptionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubscriptionLogic) GetSubscription(c echo.Context) (resp *types.Subscription, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for getting subscription")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Fetch latest non-canceled Subscription from DB
	// 2. Fetch latest non-canceled Subscription using model function
	subscriptionPtr, err := models.FindLatestActiveSubscriptionByUserID(l.svcCtx.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("No active subscription found for user %s", userID.String())
			// Return nil response, handler should interpret as 204 No Content or similar
			return nil, nil
		}
		l.Errorf("Error fetching subscription for user %s: %v", userID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve subscription details")
	}
	subscription := *subscriptionPtr // Dereference if found

	// 3. Map models.Subscription to types.Subscription
	resp = &types.Subscription{
		ID:                 subscription.ID.String(),     // Convert UUID
		UserID:             subscription.UserID.String(), // Convert UUID
		PlanID:             subscription.PlanID,          // PlanID is string
		Status:             subscription.Status,
		CurrentPeriodStart: subscription.CurrentPeriodStart.Format(time.RFC3339), // Format time
		CurrentPeriodEnd:   subscription.CurrentPeriodEnd.Format(time.RFC3339),   // Format time
		// types.Subscription doesn't include CancelAtPeriodEnd, CreatedAt, UpdatedAt, or nested Plan
	}

	l.Infof("Retrieved subscription %s for user %s", resp.ID, userID.String())
	return resp, nil
}
