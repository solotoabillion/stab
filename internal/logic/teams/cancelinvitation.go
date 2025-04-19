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

type CancelInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelInvitationLogic {
	return &CancelInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteCancelInvitation handles cancelling a pending invitation.
func (l *CancelInvitationLogic) DeleteCancelInvitation(c echo.Context, req *types.TeamInvitationRequest) (resp *types.InvitationResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for cancelling invitation")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Parse TeamID and InvitationID from request path params
	teamUUID, err := uuid.Parse(req.TeamID)
	if err != nil {
		l.Errorf("Invalid TeamID format in request: %s, error: %v", req.TeamID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}
	// Assuming Invitation ID is also a UUID in the new model
	invitationUUID, err := uuid.Parse(req.InvitationID)
	if err != nil {
		l.Errorf("Invalid InvitationID format in request: %s, error: %v", req.InvitationID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid invitation ID format")
	}

	// 3. Verify requesting user is a member and get their role using model function
	requestingMembership, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, userID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted action on team %s but is not a member", userID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team")
		}
		l.Errorf("DB error verifying membership for user %s in team %s: %v", userID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	requestingUserRole := requestingMembership.Role // Role is available from the returned membership

	// 4. Check permissions (Owner or Admin)
	isAdminOrOwner := requestingUserRole == models.RoleOwner || requestingUserRole == models.RoleAdmin
	if !isAdminOrOwner {
		l.Infof("User %s (Role: %s) attempted to cancel invitation %s for team %s without permission", userID.String(), requestingUserRole, invitationUUID.String(), teamUUID.String())
		return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to cancel invitations for this team")
	}

	// 5. Find the Invitation record using model function
	invitation, err := models.FindInvitationByIDAndTeam(l.svcCtx.DB, invitationUUID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Either ID is wrong or it doesn't belong to this team
			return nil, echo.NewHTTPError(http.StatusNotFound, "Invitation not found in this team")
		}
		l.Errorf("DB error finding invitation %s for team %s: %v", invitationUUID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find invitation")
	}

	// 6. Check if the invitation is still pending
	if invitation.Status != models.StatusPending {
		// Consider what status code is best - Bad Request or Conflict?
		return nil, echo.NewHTTPError(http.StatusConflict, "This invitation is no longer pending and cannot be cancelled")
	}

	// 7. Delete the Invitation record using model function
	err = models.DeleteInvitation(l.svcCtx.DB, invitation.ID) // Use the ID from the found invitation
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// This could happen if deleted between find and delete calls
			l.Infof("Attempted to delete invitation %s for team %s, but it was not found (possibly already deleted): %v", invitationUUID.String(), teamUUID.String(), err)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Invitation not found (or already cancelled)")
		}
		// Handle other potential errors from the delete function
		l.Errorf("Failed to delete invitation %s for team %s: %v", invitationUUID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to cancel invitation")
	}

	// 8. Return success
	l.Infof("User %s successfully cancelled invitation %s (Email: %s) for team %s", userID.String(), invitation.ID.String(), invitation.Email, teamUUID.String()) // Use invitation.ID
	// Response type expects InvitationResponse, return success message
	resp = &types.InvitationResponse{
		Success: true,
		Message: "Invitation cancelled successfully",
		// Invitation field is omitted
	}
	return resp, nil
}
