package teams

import (
	"context"
	"errors"
	"net/http"
	"time" // Added for time formatting

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ListInvitationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListInvitationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListInvitationsLogic {
	return &ListInvitationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetListInvitations handles listing pending invitations for a team.
func (l *ListInvitationsLogic) GetListInvitations(c echo.Context, req *types.TeamRequest) (resp *types.TeamInvitationsResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for listing invitations")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Parse TeamID from request path param
	teamUUID, err := uuid.Parse(req.TeamID)
	if err != nil {
		l.Errorf("Invalid TeamID format in request: %s, error: %v", req.TeamID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}

	// 3. Verify requesting user is a member and get their role
	// 3. Verify requesting user is a member using model function
	requestingMembershipPtr, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, userID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted action on team %s but is not a member", userID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team")
		}
		l.Errorf("DB error verifying membership for user %s in team %s: %v", userID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	requestingMembership := *requestingMembershipPtr // Dereference pointer
	requestingUserRole := requestingMembership.Role

	// 4. Check permissions (Owner or Admin)
	isAdminOrOwner := requestingUserRole == models.RoleOwner || requestingUserRole == models.RoleAdmin
	if !isAdminOrOwner {
		l.Infof("User %s (Role: %s) attempted to list invitations for team %s without permission", userID.String(), requestingUserRole, teamUUID.String())
		return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to view invitations for this team")
	}

	// 5. Fetch pending Invitation records for the teamID
	// 5. Fetch pending Invitation records using model function
	invitations, err := models.FindPendingInvitationsByTeam(l.svcCtx.DB, teamUUID)
	if err != nil {
		l.Errorf("Failed to retrieve pending invitations for team %s: %v", teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve invitations")
	}
	// Model function ensures empty slice

	// 6. Map models.Invitation to types.Invitation
	invitationList := make([]types.Invitation, 0, len(invitations))
	for _, inv := range invitations {
		invitationList = append(invitationList, types.Invitation{
			// ID is not in types.Invitation, only Token
			Email:     inv.Email,
			TeamID:    inv.TeamID.String(),
			Role:      string(inv.Role),
			Token:     inv.Token,
			Status:    string(inv.Status),
			ExpiresAt: inv.ExpiresAt.Format(time.RFC3339), // Use standard format
		})
	}

	// Model function ensures empty slice, no need for nil check here

	// 7. Return list of invitations
	l.Infof("Retrieved %d pending invitations for team %s for user %s", len(invitationList), teamUUID.String(), userID.String())
	resp = &types.TeamInvitationsResponse{
		Success:     true,
		Message:     "Pending invitations retrieved successfully",
		Invitations: invitationList,
	}
	return resp, nil
}
