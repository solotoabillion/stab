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

type AcceptInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAcceptInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptInvitationLogic {
	return &AcceptInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostAcceptInvitation handles accepting an invitation (requires auth).
func (l *AcceptInvitationLogic) PostAcceptInvitation(c echo.Context, req *types.InvitationTokenRequest) (resp *types.InvitationResponse, err error) {
	// 1. Get token from path parameter
	token := req.Token
	if token == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invitation token is required")
	}

	// 2. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for accepting invitation")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	acceptingUser, ok := userCtx.(*models.User)
	if !ok || acceptingUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	acceptingUserID := acceptingUser.ID
	acceptingUserEmail := acceptingUser.Email

	// 3. Accept invitation using model function, getting back membership and final invitation state
	_, finalInvitation, err := models.AcceptInvitation(l.svcCtx.DB, acceptingUserID, token)
	// The finalInvitation object is returned even if there's an error, allowing us to check its state.

	if err != nil {
		// Handle specific errors returned by AcceptInvitation
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("Accept invitation attempt failed: token %s not found", token)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Invitation not found or invalid")
		} else if err.Error() == "invitation is no longer pending" {
			l.Infof("Accept invitation attempt failed for token %s: status was %s", token, finalInvitation.Status)
			// Check if it was already accepted by this user - if so, maybe return success?
			// For now, return Bad Request as the action couldn't be performed now.
			return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation is no longer pending")
		} else if err.Error() == "invitation has expired" {
			l.Infof("Accept invitation attempt failed for token %s: expired", token)
			return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation has expired")
		} else if err.Error() == "authenticated user email does not match invited email" {
			l.Infof("User %s (%s) attempted to accept invitation with token %s meant for different email (%s)", acceptingUserID.String(), acceptingUserEmail, token, finalInvitation.Email)
			return nil, echo.NewHTTPError(http.StatusForbidden, "This invitation is not addressed to your email")
		} else if err.Error() == "failed to update invitation status, it might have been accepted or declined already" {
			// This indicates a race condition where the status changed between check and update.
			// The model function now returns the invitation state found during the transaction.
			l.Infof("Invitation %s status was not pending when trying to accept (race condition?). Final status: %s", token, finalInvitation.Status) // Changed Warnf to Infof
			// Treat as success from user perspective, as the membership was likely created.
			// Proceed to format response below using finalInvitation.
		} else {
			// Handle other potential errors (DB errors during transaction, etc.)
			l.Errorf("Failed to accept invitation for token %s: %v", token, err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to accept invitation")
		}
		// If we handled a specific error that allows proceeding (like the race condition),
		// finalInvitation will be non-nil here.
		if finalInvitation == nil {
			// Should not happen if error handling is correct, but safeguard.
			l.Errorf("Internal logic error: finalInvitation is nil after handling error for token %s", token)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error processing invitation")
		}
	}
	// If err is nil, finalInvitation is guaranteed to be non-nil and accepted.

	// 4. Check if finalInvitation is valid (should always be non-nil if we reach here)
	if finalInvitation == nil {
		// This case should ideally be unreachable due to error handling above.
		l.Errorf("Internal logic error: finalInvitation is nil after successful acceptance or handled error for token %s", token)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error processing invitation")
	}

	// 5. Log success (using details from the returned finalInvitation)
	l.Infof("User %s (%s) successfully accepted invitation %s to join team %s as %s (Final Status: %s)", acceptingUserID.String(), acceptingUserEmail, finalInvitation.ID.String(), finalInvitation.TeamID.String(), finalInvitation.Role, finalInvitation.Status)

	// 6. Map the final invitation details for the response

	// (Removed redundant logging from previous step)
	respInvitation := types.Invitation{
		Email:     finalInvitation.Email,
		TeamID:    finalInvitation.TeamID.String(),
		Role:      string(finalInvitation.Role),
		Token:     finalInvitation.Token,
		Status:    string(finalInvitation.Status), // Use actual final status
		ExpiresAt: finalInvitation.ExpiresAt.Format(time.RFC3339),
	}

	resp = &types.InvitationResponse{
		Success:    true,
		Message:    "Invitation accepted successfully",
		Invitation: respInvitation, // Include details of the accepted invitation
	}
	return resp, nil
}
