package teams

import (
	"context"
	"errors"
	"net/http"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ListMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMembersLogic {
	return &ListMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetListMembers handles listing members of a specific team.
func (l *ListMembersLogic) GetListMembers(c echo.Context, req *types.TeamRequest) (resp *types.TeamMembersResponse, err error) {
	// 1. Get authenticated UserID from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for listing team members")
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

	// 3. Verify requesting user is a member of the team (Authorization check)
	// 3. Verify requesting user is a member of the team using model function
	_, err = models.FindMembershipByUserAndTeam(l.svcCtx.DB, userID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted to list members for team %s but is not a member", userID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team's members")
		}
		l.Errorf("DB error verifying membership for user %s in team %s: %v", userID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	// If no error, user is a member

	// 4. Fetch all Memberships for the teamID
	// 4. Fetch all Memberships for the teamID using model function
	memberships, err := models.FindMembershipsByTeam(l.svcCtx.DB, teamUUID)
	if err != nil {
		l.Errorf("Failed to retrieve members for team %s: %v", teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve team members")
	}
	// Model function ensures empty slice

	// 5. Map models.Membership to types.Membership
	memberList := make([]types.Membership, 0, len(memberships))
	for _, m := range memberships {
		memberList = append(memberList, types.Membership{
			UserID: m.UserID.String(), // Convert UUID to string
			TeamID: m.TeamID.String(), // Convert UUID to string
			Role:   string(m.Role),    // Convert models.Role to string
		})
	}

	// Model function ensures empty slice, no need for nil check here

	// 6. Return list of members
	l.Infof("Retrieved %d members for team %s for user %s", len(memberList), teamUUID.String(), userID.String())
	resp = &types.TeamMembersResponse{
		Success: true,
		Message: "Team members retrieved successfully",
		Members: memberList,
	}
	return resp, nil
}
