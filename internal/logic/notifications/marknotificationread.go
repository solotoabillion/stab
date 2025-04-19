package notifications

import (
	"context" // Keep context for l.ctx
	"net/http"
	"time"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type MarkNotificationReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkNotificationReadLogic {
	return &MarkNotificationReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkNotificationReadLogic) PostMarkNotificationRead(c echo.Context, req *types.NotificationRequest) (resp *types.NotificationResponse, err error) {
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

	// 2. Get Notification ID from request and parse
	notificationID, err := uuid.Parse(req.NotificationID)
	if err != nil {
		l.Logger.Errorf("Invalid notification ID format '%s': %v", req.NotificationID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID format")
	}

	// 3. Mark notification as read using the models package function
	updatedNotification, err := models.MarkNotificationAsRead(l.svcCtx.DB, userID, notificationID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Logger.Errorf("Notification %s not found or does not belong to user %s", notificationID, userID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Notification not found or does not belong to user")
		}
		// Handle other potential errors from MarkNotificationAsRead
		l.Logger.Errorf("Failed to mark notification %s as read for user %s: %v", notificationID, userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update notification")
	}

	// Check if the notification was returned (it will be non-nil on success, including idempotent cases)
	if updatedNotification == nil {
		// This case should ideally not be reached if MarkNotificationAsRead handles errors correctly
		l.Logger.Error("MarkNotificationAsRead returned nil notification without error")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error processing notification update")
	}

	// Determine message based on whether it was already read (ReadAt might be set before the call)
	message := "Notification marked as read successfully"
	if updatedNotification.ReadAt != nil && updatedNotification.UpdatedAt.Sub(*updatedNotification.ReadAt) > time.Second { // Check if ReadAt was set before this operation
		// This check is imperfect; ideally MarkNotificationAsRead would return a flag
		// For now, we assume if it exists and is returned, it's either newly read or was already read.
		// Let's refine the message slightly if we can infer it was already read.
		// A better approach would be for the model function to return an 'already_read' boolean.
		// Re-fetching to check the state before the update is inefficient.
		// We'll stick to a generic success message for now.
		l.Logger.Infof("Notification %s was already marked as read for user %s", notificationID, userID)
		// message = "Notification was already marked as read" // Keep generic success for now
	} else {
		l.Logger.Infof("Marked notification %s as read for user %s", notificationID, userID)
	}

	// 4. Prepare response using the returned notification
	resp = &types.NotificationResponse{
		Success: true,
		Message: message, // Use determined message
		Notification: types.Notification{
			ID:        updatedNotification.ID.String(),
			UserID:    updatedNotification.UserID.String(),
			Title:     updatedNotification.Title,
			Body:      updatedNotification.Body,
			Type:      updatedNotification.Type,
			IsRead:    updatedNotification.IsRead,
			CreatedAt: updatedNotification.CreatedAt.Format(time.RFC3339),
		},
	}

	return resp, nil
}
