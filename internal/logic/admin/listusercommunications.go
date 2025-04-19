package admin

import (
	"context"
	"errors" // Added for error checking
	"net/http"
	"time" // Added for time formatting

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/google/uuid" // Added for UUID parsing
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm" // Added for gorm.ErrRecordNotFound
)

type ListUserCommunicationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUserCommunicationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUserCommunicationsLogic {
	return &ListUserCommunicationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUserCommunicationsLogic) GetListUserCommunications(c echo.Context, req *types.AdminUserRequest) (resp *types.AdminUserCommunicationsResponse, err error) {
	l.Logger.Infof("Admin request: ListUserCommunications for UserID: %s", req.UserID)

	// 1. Parse UserID from request
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		l.Logger.Errorf("Admin: Invalid UserID format: %s, error: %v", req.UserID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	// 2. Verify user exists (optional, but good practice)
	_, err = models.FindUserByID(l.svcCtx.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Logger.Infof("Admin: User not found when listing communications: %s", userID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		l.Logger.Errorf("Admin: Failed to verify user %s existence: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify user")
	}

	// 3. Fetch notifications using the existing model function
	// TODO: Add pagination/filtering if needed for admin view
	dbNotifications, err := models.FindNotificationsByUserID(l.svcCtx.DB, userID)
	if err != nil {
		l.Logger.Errorf("Admin: Failed to retrieve notifications for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve communications")
	}
	// Model function ensures empty slice

	// 4. Map models.Notification to types.Notification
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
	resp = &types.AdminUserCommunicationsResponse{
		Success:       true,
		Message:       "User communications retrieved successfully",
		Notifications: apiNotifications,
	}

	l.Logger.Infof("Admin: Retrieved %d communications for user %s", len(apiNotifications), userID)
	return resp, nil
}
