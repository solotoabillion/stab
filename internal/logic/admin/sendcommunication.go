package admin

import (
	"context"
	"errors" // Added
	"net/http"
	"time" // Added

	"stab/middleware" // Added for admin user context
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/google/uuid" // Added
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm" // Added
)

type SendCommunicationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendCommunicationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCommunicationLogic {
	return &SendCommunicationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostSendCommunication creates a notification record for a user, initiated by an admin.
// It mimics the old SendCommunication handler but uses the Notification model.
func (l *SendCommunicationLogic) PostSendCommunication(c echo.Context, req *types.SendCommunicationCombinedRequest) (resp *types.NotificationResponse, err error) { // Changed signature
	l.Logger.Infof("Admin request: SendCommunication for UserID: %s", req.UserID)

	// 1. Parse target UserID from request path (via req)
	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		l.Logger.Errorf("Admin: Invalid target UserID format: %s, error: %v", req.UserID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	// 2. Get Admin User ID from context (assuming middleware adds this)
	// Note: The original code used auth.GetUserIDFromContext(c). We adapt this.
	// Assuming the admin user model is also *models.User.
	adminUserCtx := l.ctx.Value(middleware.ContextUserKey) // Use logic context
	if adminUserCtx == nil {
		l.Logger.Error("Admin: Failed to get admin user from context")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Admin user not authenticated or context missing")
	}
	adminUser, ok := adminUserCtx.(*models.User)
	if !ok || adminUser.Role != models.SystemRoleAdmin { // Ensure the user is actually an admin
		l.Logger.Errorf("Admin: User in context is not an admin or type assertion failed. UserID: %v", adminUserCtx)
		return nil, echo.NewHTTPError(http.StatusForbidden, "Action requires admin privileges")
	}
	// adminID := adminUser.ID // We don't store admin ID in Notification model

	// 3. Verify target user exists (optional but good practice)
	targetUser, err := models.FindUserByID(l.svcCtx.DB, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Logger.Infof("Admin: Target user not found for sending communication: %s", targetUserID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Target user not found")
		}
		l.Logger.Errorf("Admin: Failed to verify target user %s existence: %v", targetUserID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify target user")
	}

	// 4. Create Notification record
	notification := models.Notification{
		UserID: targetUserID,
		Type:   req.Type, // Use type from combined request
		Title:  req.Title,
		Body:   req.Body,
		IsRead: false, // Notifications start unread
		// ReadAt will be set when marked read
	}

	// 5. Save notification using model function
	err = models.CreateNotification(l.svcCtx.DB, &notification)
	if err != nil {
		l.Logger.Errorf("Admin: Failed to create notification for user %s: %v", targetUserID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to save communication")
	}

	// 6. TODO: Trigger actual sending mechanism if Type requires it (e.g., email)
	// This would likely involve an event or calling another service.
	if req.Type == "email" { // Check type from combined request
		l.Logger.Infof("TODO: Trigger email sending for notification %s to user %s (%s)", notification.ID, targetUserID, targetUser.Email)
		// Example: l.svcCtx.EventBus.Publish("notification.created", notification)
	}

	// 7. Map created notification to response type
	respNotif := types.Notification{
		ID:        notification.ID.String(),
		UserID:    notification.UserID.String(),
		Title:     notification.Title,
		Body:      notification.Body,
		Type:      notification.Type,
		IsRead:    notification.IsRead,
		CreatedAt: notification.CreatedAt.Format(time.RFC3339),
	}

	// 8. Prepare response
	resp = &types.NotificationResponse{
		Success:      true,
		Message:      "Communication sent successfully",
		Notification: respNotif,
	}

	l.Logger.Infof("Admin: Successfully created communication (Notification ID: %s) for user %s by admin %s", notification.ID, targetUserID, adminUser.ID)
	return resp, nil
}
