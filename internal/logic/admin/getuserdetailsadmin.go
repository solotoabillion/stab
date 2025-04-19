package admin

import (
	"context"
	"encoding/json" // Added for JSON handling
	"errors"        // Added for error checking
	"net/http"
	"time" // Added for time formatting

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	// Added for shared utils
	"github.com/google/uuid" // Added for UUID parsing
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm" // Added for gorm.ErrRecordNotFound
)

type GetUserDetailsAdminLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserDetailsAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserDetailsAdminLogic {
	return &GetUserDetailsAdminLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserDetailsAdminLogic) GetUserDetailsAdmin(c echo.Context, req *types.AdminUserRequest) (resp *types.AdminUserResponse, err error) {
	l.Logger.Infof("Admin request: GetUserDetailsAdmin for UserID: %s", req.UserID)

	// 1. Parse UserID from request
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		l.Logger.Errorf("Admin: Invalid UserID format: %s, error: %v", req.UserID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	// 2. Fetch user details using model function
	user, err := models.FindUserWithDetailsForAdmin(l.svcCtx.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Logger.Infof("Admin: User not found: %s", userID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		l.Logger.Errorf("Admin: Failed to retrieve details for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve user details")
	}

	// 3. Unmarshal ProfileData JSON
	var profileData types.UserProfileData
	if user.ProfileData != nil {
		if err := json.Unmarshal(user.ProfileData, &profileData); err != nil {
			l.Logger.Errorf("Admin: Error unmarshalling profile data for user %s: %v", userID, err)
			// Decide if this is a server error or if we should return partial data
			// For admin details, maybe return error? Or just log and continue? Let's log and continue for now.
			// profileData will remain empty struct
		}
	}

	// 4. Map models.User to types.User (for the response)
	respUser := types.User{
		ID:               user.ID.String(),
		Email:            user.Email,
		ProfileData:      profileData, // Assign unmarshalled data
		Role:             string(user.Role),
		ApiKey:           user.ApiKey,
		DefaultSubdomain: user.DefaultSubdomain,
		AccountStatus:    string(user.AccountStatus),
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        user.UpdatedAt.Format(time.RFC3339),
	}

	// 5. Prepare response
	resp = &types.AdminUserResponse{
		Success: true,
		Message: "User details retrieved successfully",
		User:    respUser,
	}

	l.Logger.Infof("Admin: Successfully retrieved details for user %s", userID)
	return resp, nil
}

// Removed local derefString helper function
