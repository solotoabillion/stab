package teams

import (
	"context"
	"net/http" // Added for status codes

	"github.com/solotoabillion/stab/middleware" // Added for ContextUserKey
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	// Added for UUID parsing/handling
	"github.com/labstack/echo/v4" // Still needed for echo.Context and HTTPError
	"github.com/zeromicro/go-zero/core/logx"
	// Removed gorm import as transaction is now in model
)

type CreateTeamLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTeamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTeamLogic {
	return &CreateTeamLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostCreateTeam handles creating a new team and the owner's membership.
// Note: The request type in the signature is *types.Team, but the API definition
// likely only expects 'name' in the request body for team creation.
// Go-zero handles binding based on the API definition. We'll use req.Name.
func (l *CreateTeamLogic) PostCreateTeam(c echo.Context, req *types.Team) (resp *types.TeamResponse, err error) {
	// 1. Get authenticated UserID from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for team creation")
		// Using echo.NewHTTPError for now, consider standardizing error responses later
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID // userID is uuid.UUID

	// Basic validation for name (go-zero validation happens earlier based on API def)
	if req.Name == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Team name is required")
	}

	// 2. Create Team and Membership using model function
	newTeam, createErr := models.CreateTeamWithOwner(l.svcCtx.DB, req.Name, userID)
	if createErr != nil {
		// The model function handles the transaction and returns a specific error
		l.Errorf("Failed to create team '%s' for user %s: %v", req.Name, userID.String(), createErr)
		// Map the error to an appropriate HTTP status code
		// TODO: Consider more specific error handling based on createErr type if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create team and membership")
	}

	// 3. Prepare and return response
	l.Infof("Team '%s' (ID: %s) created successfully by user %s", newTeam.Name, newTeam.ID.String(), userID.String())
	resp = &types.TeamResponse{
		Success: true,
		Message: "Team created successfully",
		Team: types.Team{ // Map models.Team to types.Team
			ID:      newTeam.ID.String(), // Convert UUID to string
			Name:    newTeam.Name,
			OwnerID: newTeam.OwnerID.String(), // Convert UUID to string
		},
	}
	return resp, nil
}
