package notifications

import (
	"context" // Keep context for l.ctx
	"net/http"
	"time"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type MarkAllNotificationsReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkAllNotificationsReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllNotificationsReadLogic {
	return &MarkAllNotificationsReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkAllNotificationsReadLogic) PostMarkAllNotificationsRead(c echo.Context) (resp *types.NotificationsResponse, err error) {
	// 1. Get User from context
	userCtx := l.ctx.Value(middleware.ContextUserKey)
	if userCtx == nil {
		l.Logger.Error("Failed to get user from context: context key not found")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	user, ok := userCtx.(*models.User)
	if !ok {
		l.Logger.Error("Failed to assert user type from context")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error: invalid user context")
	}
	userID := user.ID // Get the user ID from the user model

	// 2. Mark all notifications as read using model function
	rowsAffected, err := models.MarkAllNotificationsAsRead(l.svcCtx.DB, userID)
	if err != nil {
		l.Logger.Errorf("Failed to mark all notifications as read for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update notifications")
	}

	l.Logger.Infof("Marked %d notifications as read for user %s", rowsAffected, userID)

	// 3. Fetch the updated notifications (API spec requires returning them)
	// 3. Fetch the updated notifications using model function
	dbNotifications, err := models.FindNotificationsByUserID(l.svcCtx.DB, userID)
	if err != nil {
		l.Logger.Errorf("Failed to retrieve notifications after marking all as read for user %s: %v", userID, err)
		// Return success but indicate potential inconsistency
		return &types.NotificationsResponse{
			Success:       true, // Operation to mark as read succeeded
			Message:       "Successfully marked all notifications as read, but failed to retrieve updated list.",
			Notifications: []types.Notification{},
		}, nil
	}
	// Model function ensures empty slice

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
		Message:       "All notifications marked as read successfully",
		Notifications: apiNotifications,
	}

	return resp, nil
}
