package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProfileLogic {
	return &ProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProfileLogic) GetProfile(c echo.Context) (resp *types.User, err error) {
	// 1. Get user from context (set by UserGuardMiddleware)
	user := session.UserFromContext(c)
	if user == nil {
		l.Errorf("User not found in context")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	// 2. Unmarshal ProfileData JSON
	var profileData types.UserProfileData
	if user.ProfileData != nil {
		if err := json.Unmarshal(user.ProfileData, &profileData); err != nil {
			l.Errorf("Error unmarshalling profile data for user %s: %v", user.Email, err)
			// Decide if this is a server error or if we should return partial data
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error processing user profile data")
		}
	}

	// 3. Prepare response (map models.User to types.User)
	resp = &types.User{
		ID:               user.ID.String(), // Convert UUID to string
		Email:            user.Email,
		ProfileData:      profileData, // Assign the unmarshalled struct
		Role:             string(user.Role),
		ApiKey:           user.ApiKey,
		DefaultSubdomain: user.DefaultSubdomain,
		AccountStatus:    string(user.AccountStatus),
		CreatedAt:        user.CreatedAt.Format(time.RFC3339), // Format time
		UpdatedAt:        user.UpdatedAt.Format(time.RFC3339), // Format time
	}

	l.Infof("Successfully retrieved profile for user %s", user.Email)
	return resp, nil
}
