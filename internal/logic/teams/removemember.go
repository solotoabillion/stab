package teams

import (
	"context"
	"errors"
	"net/http"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RemoveMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveMemberLogic {
	return &RemoveMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteRemoveMember handles removing a member from a team.
func (l *RemoveMemberLogic) DeleteRemoveMember(c echo.Context, req *types.TeamMemberRequest) (resp *types.TeamMemberResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for removing team member")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	requestingUserID := authedUser.ID

	// 2. Parse TeamID and MemberID from request
	teamUUID, err := uuid.Parse(req.TeamID)
	if err != nil {
		l.Errorf("Invalid TeamID format in request: %s, error: %v", req.TeamID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}
	memberUUIDToRemove, err := uuid.Parse(req.MemberID)
	if err != nil {
		l.Errorf("Invalid MemberID format in request: %s, error: %v", req.MemberID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid member ID format")
	}

	// 3. Verify requesting user is a member and get their role using model function
	requestingMembership, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, requestingUserID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted action on team %s but is not a member", requestingUserID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team")
		}
		l.Errorf("DB error verifying membership for user %s in team %s: %v", requestingUserID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	requestingUserRole := requestingMembership.Role // Role is available from the returned membership

	// 4. Check permissions (Owner or Admin)
	isAdminOrOwner := requestingUserRole == models.RoleOwner || requestingUserRole == models.RoleAdmin
	if !isAdminOrOwner {
		l.Infof("User %s (Role: %s) attempted to remove member %s from team %s without permission", requestingUserID.String(), requestingUserRole, memberUUIDToRemove.String(), teamUUID.String())
		return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to remove members from this team")
	}

	// 5. Prevent users from removing themselves
	if requestingUserID == memberUUIDToRemove {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "You cannot remove yourself from the team using this endpoint")
	}

	// 6. Find the membership record of the user to be removed using model function
	membershipToRemove, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, memberUUIDToRemove, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "Member not found in this team")
		}
		l.Errorf("DB error finding membership for user %s in team %s: %v", memberUUIDToRemove.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find member")
	}

	// 7. Verify target member is not the owner
	if membershipToRemove.Role == models.RoleOwner {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "The team owner cannot be removed")
	}

	// 8. Delete the Membership record using model function
	err = models.DeleteMembershipByUserAndTeam(l.svcCtx.DB, memberUUIDToRemove, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// This could mean the member was already removed between the find and delete calls
			l.Infof("Attempted to delete membership for user %s in team %s, but it was not found (possibly already removed): %v", memberUUIDToRemove.String(), teamUUID.String(), err)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Member not found (or already removed)")
		}
		// Handle other potential errors from the delete function
		l.Errorf("Failed to delete membership for user %s in team %s: %v", memberUUIDToRemove.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove member")
	}

	// 9. Return success
	l.Infof("User %s successfully removed member %s from team %s", requestingUserID.String(), memberUUIDToRemove.String(), teamUUID.String())
	// The signature expects TeamMemberResponse, but we don't have member details to return.
	// Return a success message.
	resp = &types.TeamMemberResponse{
		Success: true,
		Message: "Member removed successfully",
		// Member field is omitted as the member is removed.
	}
	return resp, nil
}
