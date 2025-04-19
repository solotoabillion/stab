package teams

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/solotoabillion/stab/middleware"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeclineInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeclineInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeclineInvitationLogic {
	return &DeclineInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostDeclineInvitation handles declining an invitation (requires auth).
func (l *DeclineInvitationLogic) PostDeclineInvitation(c echo.Context, req *types.InvitationTokenRequest) (resp *types.InvitationResponse, err error) {
	// 1. Get token from path parameter
	token := req.Token
	if token == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invitation token is required")
	}

	// 2. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for declining invitation")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	decliningUser, ok := userCtx.(*models.User)
	if !ok || decliningUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	decliningUserID := decliningUser.ID
	decliningUserEmail := decliningUser.Email

	// 3. Find Invitation by token using model function
	// We don't strictly need the Team details here, but FindInvitationByTokenWithTeam is available.
	// If a simpler FindInvitationByToken existed, we could use that.
	invitation, err := models.FindInvitationByTokenWithTeam(l.svcCtx.DB, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("Decline invitation attempt failed: token %s not found", token)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Invitation not found or invalid")
		}
		l.Errorf("DB error finding invitation by token %s: %v", token, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve invitation")
	}
	// invitation is now a pointer, need to dereference or use ->

	// 4. Validate invitation status, expiry, and email match
	if invitation.Status != models.StatusPending {
		l.Infof("Decline invitation attempt failed for token %s: status is %s", token, invitation.Status)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation is no longer pending")
	}
	if time.Now().After(invitation.ExpiresAt) {
		l.Infof("Decline invitation attempt failed for token %s: expired at %s", token, invitation.ExpiresAt.String())
		// Optionally update status to Expired in DB here
		// l.svcCtx.DB.Model(&invitation).Update("status", models.StatusExpired)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation has expired")
	}
	if invitation.Email != decliningUserEmail {
		l.Infof("User %s (%s) attempted to decline invitation %s meant for %s", decliningUserID.String(), decliningUserEmail, invitation.ID.String(), invitation.Email) // Used Infof here
		return nil, echo.NewHTTPError(http.StatusForbidden, "This invitation is not addressed to your email")
	}

	// 5. Update Invitation status to Declined using model function
	updatedInvitation, err := models.UpdateInvitationStatus(l.svcCtx.DB, invitation.ID, models.StatusDeclined)
	if err != nil {
		// Handle specific errors from UpdateInvitationStatus
		if err.Error() == "invitation is no longer pending" {
			l.Infof("Invitation %s status was not pending when trying to decline (race condition?). Current status: %s", invitation.ID.String(), updatedInvitation.Status)
			return nil, echo.NewHTTPError(http.StatusConflict, "Invitation was already accepted or cancelled")
		} else if err.Error() == "invitation status changed unexpectedly" {
			l.Infof("Invitation %s status changed unexpectedly during decline attempt. Final status: %s", invitation.ID.String(), updatedInvitation.Status) // Changed Warnf to Infof
			// Treat as conflict, as the decline didn't happen as expected.
			return nil, echo.NewHTTPError(http.StatusConflict, "Invitation status changed unexpectedly")
		}
		// Handle other potential errors (DB errors, etc.)
		l.Errorf("Failed to update invitation %s status to declined: %v", invitation.ID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update invitation status")
	}
	// Use the returned updatedInvitation for the response
	invitation = updatedInvitation // Update local variable to the returned state

	// 6. Return success
	l.Infof("User %s (%s) successfully declined invitation %s to join team %s", decliningUserID.String(), decliningUserEmail, invitation.ID.String(), invitation.TeamID.String())

	// Map the updated invitation details for the response
	respInvitation := types.Invitation{
		Email:     invitation.Email,
		TeamID:    invitation.TeamID.String(),
		Role:      string(invitation.Role),
		Token:     invitation.Token,
		Status:    string(invitation.Status), // Use status from the returned invitation object
		ExpiresAt: invitation.ExpiresAt.Format(time.RFC3339),
	}

	resp = &types.InvitationResponse{
		Success:    true,
		Message:    "Invitation declined successfully",
		Invitation: respInvitation, // Include details of the declined invitation
	}
	return resp, nil
}
