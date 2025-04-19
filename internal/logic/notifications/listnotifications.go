package notifications

import (
	"context" // Keep context for l.ctx
	"net/http"
	"time"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListNotificationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNotificationsLogic {
	return &ListNotificationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListNotificationsLogic) GetListNotifications(c echo.Context) (resp *types.NotificationsResponse, err error) {
	// 1. Get User from context using the logic's context
	userCtx := l.ctx.Value(middleware.ContextUserKey)
	if userCtx == nil {
		l.Logger.Error("Failed to get user from context: context key not found")
		// Use echo's error handling
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	user, ok := userCtx.(*models.User)
	if !ok {
		l.Logger.Error("Failed to assert user type from context")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error: invalid user context")
	}
	userID := user.ID // Get the user ID from the user model

	// 2. Query Params (Filtering/Pagination - Optional, not defined in API spec for this endpoint yet)
	// For now, fetch all notifications for the user. Add pagination/filtering later if needed.

	// 3. Fetch notifications using the models package function
	dbNotifications, err := models.FindNotificationsByUserID(l.svcCtx.DB, userID)
	if err != nil {
		l.Logger.Errorf("Failed to retrieve notifications for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve notifications")
	}
	// The model function already ensures an empty slice is returned if no results

	// 4. Map DB models to API types
	apiNotifications := make([]types.Notification, len(dbNotifications))
	for i, n := range dbNotifications {
		apiNotifications[i] = types.Notification{
			ID:        n.ID.String(),
			UserID:    n.UserID.String(),
			Title:     n.Title,
			Body:      n.Body,
			Type:      n.Type,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		}
	}

	// 5. Prepare response
	resp = &types.NotificationsResponse{
		Success:       true,
		Message:       "Notifications retrieved successfully",
		Notifications: apiNotifications,
	}

	return resp, nil
}
