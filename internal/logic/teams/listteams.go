package teams

import (
	"context"
	"net/http" // Added for status codes

	"github.com/solotoabillion/stab/middleware" // Added for ContextUserKey
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"      // Added for uuid.Nil check
	"github.com/labstack/echo/v4" // Still needed for echo.Context and HTTPError
	"github.com/zeromicro/go-zero/core/logx"
)

type ListTeamsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTeamsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTeamsLogic {
	return &ListTeamsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetListTeams handles listing teams the authenticated user is a member of.
// The request parameter 'c echo.Context' is provided by the handler.
func (l *ListTeamsLogic) GetListTeams(c echo.Context) (resp *[]types.Team, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for listing teams")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Find all Memberships for the UserID, preloading the Team data
	var memberships []models.Membership
	result := l.svcCtx.DB.Preload("Team").Where("user_id = ?", userID).Find(&memberships)
	if result.Error != nil {
		l.Errorf("Failed to retrieve memberships for user %s: %v", userID.String(), result.Error)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve teams")
	}

	// 3. Extract the Team objects and map to types.Team
	teamsList := make([]types.Team, 0, len(memberships))
	for _, m := range memberships {
		// Ensure Team data was actually loaded
		if m.Team.ID != uuid.Nil { // Check against nil UUID
			teamsList = append(teamsList, types.Team{
				ID:      m.Team.ID.String(), // Convert UUID to string
				Name:    m.Team.Name,
				OwnerID: m.Team.OwnerID.String(), // Convert UUID to string
			})
		} else {
			// This indicates a potential data integrity issue or preload failure
			l.Errorf("[Warning] Membership found (ID: %d) for user %s but associated team (ID: %s) data is missing or invalid.", m.ID, userID.String(), m.TeamID.String())
		}
	}

	// 4. Return list of teams
	l.Infof("Retrieved %d teams for user %s", len(teamsList), userID.String())
	resp = &teamsList // Assign the populated slice to the response pointer
	return resp, nil
}
