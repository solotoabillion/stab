package teams

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid" // Needed for Team.ID check
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetInvitationDetailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInvitationDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInvitationDetailsLogic {
	return &GetInvitationDetailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetInvitationDetails handles fetching invitation details using the token (public).
func (l *GetInvitationDetailsLogic) GetInvitationDetails(c echo.Context, req *types.InvitationTokenRequest) (resp *types.InvitationResponse, err error) {
	// 1. Get token from path parameter
	token := req.Token
	if token == "" {
		// This validation might be handled by go-zero based on API definition
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invitation token is required")
	}

	// 2. Find Invitation by token, preloading Team details for context
	// 2. Find Invitation by token using model function
	invitationPtr, err := models.FindInvitationByTokenWithTeam(l.svcCtx.DB, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("Invitation details requested for invalid/unknown token: %s", token)
			return nil, echo.NewHTTPError(http.StatusNotFound, "Invitation not found or invalid")
		}
		l.Errorf("DB error finding invitation by token %s: %v", token, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve invitation details")
	}
	// Dereference pointer as the rest of the logic expects the struct value
	invitation := *invitationPtr

	// 3. Check status
	if invitation.Status != models.StatusPending {
		l.Infof("Invitation details requested for token %s, but status is %s", token, invitation.Status)
		// Return specific status - e.g., Gone or Bad Request? Using Bad Request for now.
		return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation is no longer pending")
	}

	// 4. Check expiry
	if time.Now().After(invitation.ExpiresAt) {
		l.Infof("Invitation details requested for token %s, but it has expired at %s", token, invitation.ExpiresAt.String())
		// Optionally update status to Expired in DB here before returning error
		// l.svcCtx.DB.Model(&invitation).Update("status", models.StatusExpired)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "This invitation has expired")
	}

	// 5. Map invitation details to response type
	teamName := "Unknown Team" // Default if preload fails
	if invitation.Team.ID != uuid.Nil {
		teamName = invitation.Team.Name
	} else {
		l.Errorf("[Warning] Invitation %s found for token %s, but associated team %s data is missing.", invitation.ID.String(), token, invitation.TeamID.String())
	}

	respInvitation := types.Invitation{
		Email:     invitation.Email,
		TeamID:    invitation.TeamID.String(),
		Role:      string(invitation.Role),
		Token:     invitation.Token, // Include token for potential frontend use
		Status:    string(invitation.Status),
		ExpiresAt: invitation.ExpiresAt.Format(time.RFC3339),
		// Note: Team Name is not part of the standard types.Invitation struct
	}

	// 6. Return success
	l.Infof("Retrieved details for invitation token %s (Team: %s, Email: %s)", token, teamName, invitation.Email)
	resp = &types.InvitationResponse{
		Success:    true,
		Message:    "Invitation details retrieved successfully",
		Invitation: respInvitation,
	}
	return resp, nil
}
