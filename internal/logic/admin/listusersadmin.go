package admin

import (
	"context"
	"encoding/json" // Added for JSON handling
	"net/http"      // Added for HTTP status codes
	"time"          // Added for time formatting

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	// Added for shared utils
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsersAdminLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUsersAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsersAdminLogic {
	return &ListUsersAdminLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUsersAdminLogic) GetListUsersAdmin(c echo.Context) (resp *[]types.User, err error) {
	l.Logger.Info("Admin request: ListUsersAdmin")

	// 1. Fetch users using model function
	// TODO: Add pagination/filtering parameters to model function call later
	modelUsers, err := models.FindAllUsersForAdmin(l.svcCtx.DB)
	if err != nil {
		l.Logger.Errorf("Admin: Failed to retrieve users: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve users")
	}

	// 2. Map models.User to types.User
	typeUsers := make([]types.User, 0, len(modelUsers))
	for _, u := range modelUsers {
		// Unmarshal ProfileData JSON
		var profileData types.UserProfileData
		if u.ProfileData != nil {
			if err := json.Unmarshal(u.ProfileData, &profileData); err != nil {
				l.Logger.Errorf("Admin: Error unmarshalling profile data for user %s: %v. Skipping profile data for this user.", u.Email, err)
				// Continue with other fields, profileData will be empty struct
			}
		}

		// Map fields available in both model and type
		user := types.User{
			ID:               u.ID.String(),
			Email:            u.Email,
			ProfileData:      profileData, // Assign unmarshalled data
			Role:             string(u.Role),
			ApiKey:           u.ApiKey,
			DefaultSubdomain: u.DefaultSubdomain,
			AccountStatus:    string(u.AccountStatus),
			CreatedAt:        u.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        u.UpdatedAt.Format(time.RFC3339),
		}
		typeUsers = append(typeUsers, user)
	}

	// Model function ensures empty slice, no need for nil check

	l.Logger.Infof("Admin: Retrieved %d users", len(typeUsers))
	resp = &typeUsers
	return resp, nil
}

// Removed local derefString helper function
