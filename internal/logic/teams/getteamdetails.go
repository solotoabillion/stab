package teams

import (
	"context"
	"errors" // Added for error checking
	"net/http"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm" // Added for gorm.ErrRecordNotFound
)

type GetTeamDetailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTeamDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTeamDetailsLogic {
	return &GetTeamDetailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTeamDetails handles fetching details for a specific team the user is a member of.
func (l *GetTeamDetailsLogic) GetTeamDetails(c echo.Context, req *types.TeamRequest) (resp *types.TeamResponse, err error) {
	// 1. Get authenticated UserID from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for getting team details")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Parse TeamID from request
	teamUUID, err := uuid.Parse(req.TeamID)
	if err != nil {
		l.Errorf("Invalid TeamID format in request: %s, error: %v", req.TeamID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}

	// 3. Verify user is a member of the team
	// 3. Verify user is a member of the team using model function
	_, err = models.FindMembershipByUserAndTeam(l.svcCtx.DB, userID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted to get details for team %s but is not a member", userID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team")
		}
		// Handle other potential database errors
		l.Errorf("DB error verifying membership for user %s in team %s: %v", userID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	// If no error, user is a member

	// 4. Fetch Team details
	// 4. Fetch Team details using model function
	teamPtr, err := models.FindTeamByID(l.svcCtx.DB, teamUUID)
	if err != nil {
		// This case is less likely if membership exists, but handle defensively
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Errorf("Membership exists for user %s in team %s, but team record not found", userID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Team data inconsistency")
		}
		l.Errorf("Failed to retrieve team details for team %s: %v", teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve team details")
	}
	team := *teamPtr // Dereference

	// 5. Prepare and return response
	l.Infof("Retrieved details for team %s for user %s", teamUUID.String(), userID.String())
	resp = &types.TeamResponse{
		Success: true,
		Message: "Team details retrieved successfully",
		Team: types.Team{
			ID:      team.ID.String(),
			Name:    team.Name,
			OwnerID: team.OwnerID.String(),
		},
	}
	return resp, nil
}
