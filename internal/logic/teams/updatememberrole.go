package teams

import (
	"context"
	"errors"
	"fmt" // Added for error formatting
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

type UpdateMemberRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateMemberRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMemberRoleLogic {
	return &UpdateMemberRoleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PatchUpdateMemberRole handles changing a member's role within a team.
// TODO: Verify that the 'req *types.TeamMemberRequest' correctly includes the 'Role' field
// from the request body, as defined in the corresponding .api file route.
// If not, the .api definition or the generated handler might need adjustment.
func (l *UpdateMemberRoleLogic) PatchUpdateMemberRole(c echo.Context, req *types.UpdateMemberRoleRequest) (resp *types.TeamMemberResponse, err error) { // Changed signature
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for updating member role")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	requestingUserID := authedUser.ID

	// 2. Parse TeamID and MemberID from path parameters using Echo context
	teamIDStr := c.Param("teamId") // Get from path param
	teamUUID, err := uuid.Parse(teamIDStr)
	if err != nil {
		l.Errorf("Invalid TeamID format in path parameter: %s, error: %v", teamIDStr, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}
	memberIDStr := c.Param("memberId") // Get from path param
	memberUUIDToUpdate, err := uuid.Parse(memberIDStr)
	if err != nil {
		l.Errorf("Invalid MemberID format in path parameter: %s, error: %v", memberIDStr, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid member ID format")
	}
	// 3. Get the new role from the request body object (`req`)
	newRoleStr := req.Role // Get role from the request body struct
	// Basic validation (framework should handle `oneof` tag)
	if newRoleStr != string(models.RoleAdmin) && newRoleStr != string(models.RoleMember) {
		// This check might be redundant if framework validation works, but good for safety.
		err := fmt.Errorf("invalid role specified: '%s'. Must be 'admin' or 'member'", newRoleStr)
		l.Error(err.Error())
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	newRole := models.Role(newRoleStr)

	// 4. Verify requesting user is a member and get their role using model function
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

	// 5. Check permissions (Owner or Admin)
	isAdminOrOwner := requestingUserRole == models.RoleOwner || requestingUserRole == models.RoleAdmin
	if !isAdminOrOwner {
		l.Infof("User %s (Role: %s) attempted to update role for member %s in team %s without permission", requestingUserID.String(), requestingUserRole, memberUUIDToUpdate.String(), teamUUID.String())
		return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to update roles in this team")
	}

	// 6. Prevent users from changing their own role
	if requestingUserID == memberUUIDToUpdate {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "You cannot change your own role using this endpoint")
	}

	// 7. Find the membership record of the user whose role is being updated using model function
	membershipToUpdate, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, memberUUIDToUpdate, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "Member not found in this team")
		}
		l.Errorf("DB error finding membership for user %s in team %s: %v", memberUUIDToUpdate.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find member")
	}

	// 8. Verify target member is not the owner
	if membershipToUpdate.Role == models.RoleOwner {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "The team owner's role cannot be changed")
	}

	// 9. Update the Membership record with the new role using model function
	err = models.UpdateMembershipRole(l.svcCtx.DB, membershipToUpdate.ID, newRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// This could happen if the membership was deleted between find and update
			l.Infof("Attempted to update role for membership ID %s (User: %s, Team: %s), but it was not found: %v", membershipToUpdate.ID, memberUUIDToUpdate, teamUUID, err) // Changed Warnf to Infof
			return nil, echo.NewHTTPError(http.StatusNotFound, "Member not found (or was recently removed)")
		}
		// Handle other potential errors from the update function (e.g., invalid role if validation moved there)
		l.Errorf("Failed to update role for user %s (Membership ID: %s) in team %s: %v", memberUUIDToUpdate.String(), membershipToUpdate.ID, teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update member role")
	}
	// Refresh the role in the local variable after successful update
	membershipToUpdate.Role = newRole

	// 10. Map updated membership to response type
	updatedMember := types.Membership{
		UserID: membershipToUpdate.UserID.String(),
		TeamID: membershipToUpdate.TeamID.String(),
		Role:   string(membershipToUpdate.Role),
	}

	// 11. Return success with the updated membership
	l.Infof("User %s successfully updated role for member %s in team %s to %s", requestingUserID.String(), memberUUIDToUpdate.String(), teamUUID.String(), newRole)
	resp = &types.TeamMemberResponse{
		Success: true,
		Message: "Member role updated successfully",
		Member:  updatedMember, // Include the updated member details
	}
	return resp, nil
}
