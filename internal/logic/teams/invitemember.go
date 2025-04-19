package teams

import (
	"context"
	"errors"
	"fmt"
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

type InviteMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInviteMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteMemberLogic {
	return &InviteMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostInviteMember handles sending an invitation to join a team.
// TODO: Verify that 'req *types.TeamRequest' correctly includes 'Email' and 'Role' fields
// from the request body, as defined in the corresponding .api file route.
// If not, the .api definition or the generated handler might need adjustment to use a different request type.
func (l *InviteMemberLogic) PostInviteMember(c echo.Context, req *types.InviteMemberRequest) (resp *types.InvitationResponse, err error) { // Changed request type to InviteMemberRequest
	// 1. Get authenticated User from context (Inviter)
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for inviting member")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	inviterID := authedUser.ID

	// 2. Parse TeamID from request path param
	teamUUID, err := uuid.Parse(req.TeamID)
	if err != nil {
		l.Errorf("Invalid TeamID format in request: %s, error: %v", req.TeamID, err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid team ID format")
	}

	// 3. Get Email and Role from request body (assuming they are in req)
	// TODO: Accessing req.Email and req.Role directly. Confirm this matches the actual request structure.
	invitedEmail := req.Email  // Assuming Email field exists
	invitedRoleStr := req.Role // Assuming Role field exists

	// Basic validation (go-zero validation should handle more)
	if invitedEmail == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invited email is required")
	}
	if invitedRoleStr != string(models.RoleAdmin) && invitedRoleStr != string(models.RoleMember) {
		err := fmt.Errorf("invalid role specified: %s. Must be 'admin' or 'member'", invitedRoleStr)
		l.Error(err.Error())
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	invitedRole := models.Role(invitedRoleStr)

	// 4. Verify inviter is a member and get their role
	// 4. Verify inviter is a member using model function
	inviterMembershipPtr, err := models.FindMembershipByUserAndTeam(l.svcCtx.DB, inviterID, teamUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("User %s attempted action on team %s but is not a member", inviterID.String(), teamUUID.String())
			return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to access this team")
		}
		l.Errorf("DB error verifying membership for user %s in team %s: %v", inviterID.String(), teamUUID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify team membership")
	}
	inviterMembership := *inviterMembershipPtr // Dereference
	inviterRole := inviterMembership.Role

	// 5. Check inviter permissions (Owner or Admin)
	isAdminOrOwner := inviterRole == models.RoleOwner || inviterRole == models.RoleAdmin
	if !isAdminOrOwner {
		l.Infof("User %s (Role: %s) attempted to invite member to team %s without permission", inviterID.String(), inviterRole, teamUUID.String())
		return nil, echo.NewHTTPError(http.StatusForbidden, "You do not have permission to invite members to this team")
	}

	// 6. Check if the invited email belongs to an existing user and if they are already a member
	// 6. Check if the invited email belongs to an existing user using model function
	invitedUserPtr, userCheckErr := models.FindUserByEmail(l.svcCtx.DB, invitedEmail)
	if userCheckErr == nil { // User exists
		invitedUser := *invitedUserPtr // Dereference
		// Check if they are already a member of this team using model function
		_, existingMemErr := models.FindMembershipByUserAndTeam(l.svcCtx.DB, invitedUser.ID, teamUUID)
		if existingMemErr == nil {
			// User is already a member
			return nil, echo.NewHTTPError(http.StatusConflict, "This user is already a member of the team")
		} else if !errors.Is(existingMemErr, gorm.ErrRecordNotFound) {
			// Database error checking membership
			l.Errorf("DB error checking existing membership for user %s in team %s: %v", invitedUser.ID.String(), teamUUID.String(), existingMemErr)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check existing membership")
		}
		// If ErrRecordNotFound, proceed (user exists but is not a member)
	} else if !errors.Is(userCheckErr, gorm.ErrRecordNotFound) {
		// Database error finding user by email
		l.Errorf("DB error checking invited user email %s: %v", invitedEmail, userCheckErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check invited user")
	}
	// If ErrRecordNotFound, proceed (user does not exist yet)

	// 7. Check for existing *pending* invitation for this email/team
	// 7. Check for existing *pending* invitation using model function
	_, existingInviteErr := models.FindPendingInvitationByTeamAndEmail(l.svcCtx.DB, teamUUID, invitedEmail)
	if existingInviteErr == nil {
		// Pending invitation already exists
		// TODO: Optionally resend the invitation email here or just return conflict
		return nil, echo.NewHTTPError(http.StatusConflict, "An invitation for this email address is already pending")
	} else if !errors.Is(existingInviteErr, gorm.ErrRecordNotFound) {
		// Database error checking invitations
		l.Errorf("DB error checking existing invitations for email %s in team %s: %v", invitedEmail, teamUUID.String(), existingInviteErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check existing invitations")
	}
	// If ErrRecordNotFound, proceed

	// 8. Create Invitation record
	invitation := models.Invitation{
		Email:     invitedEmail,
		TeamID:    teamUUID, // Use parsed UUID
		InviterID: inviterID,
		Role:      invitedRole,
		Status:    models.StatusPending, // Explicitly set initial status
		// Token and ExpiresAt should be set by BeforeCreate hook in models/invitation.go
	}
	// Create using model function (relies on BeforeCreate hook)
	err = models.CreateInvitation(l.svcCtx.DB, &invitation)
	if err != nil {
		l.Errorf("Failed to create invitation record for email %s to team %s by user %s: %v", invitedEmail, teamUUID.String(), inviterID.String(), err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create invitation")
	}

	// 9. Send email with invitation link (Commented out - requires communication service setup)
	/*
		frontendURL := os.Getenv("FRONTEND_URL") // TODO: Get from config svcCtx.Config.FrontendURL
		if frontendURL == "" {
			frontendURL = "http://localhost:5173" // Fallback for local dev
			l.Warnf("FRONTEND_URL not set, using default for invitation link.")
		}
		inviteLink := fmt.Sprintf("%s/accept-invite/%s", frontendURL, invitation.Token)
		emailSubject := "You're Invited to Join a Team!"
		// TODO: Include inviter name/email and team name in the body
		emailBody := fmt.Sprintf("You have been invited to join a team as a %s.\n\nClick the link below to accept:\n\n%s\n\nThis link will expire on %s.",
			invitation.Role,
			inviteLink,
			invitation.ExpiresAt.Format(time.RFC1123)) // Format expiry time nicely

		// TODO: Replace with actual call to communication service via svcCtx if available
		// go func() {
		// 	err := communication.SendMessage(l.ctx, invitation.Email, emailSubject, emailBody, communication.CommunicationChannelEmail)
		// 	if err != nil {
		// 		l.Errorf("Failed to send invitation email to %s for team %s: %v", invitation.Email, teamUUID.String(), err)
		// 	}
		// }()
	*/
	l.Infof("Invitation created for %s to join team %s (Role: %s) by user %s. Token: %s", invitedEmail, teamUUID.String(), invitedRole, inviterID.String(), invitation.Token)

	// 10. Map created invitation to response type
	respInvitation := types.Invitation{
		// ID is not typically part of the response for creating an invite
		Email:     invitation.Email,
		TeamID:    invitation.TeamID.String(),
		Role:      string(invitation.Role),
		Token:     invitation.Token, // Return token so frontend/user knows it
		Status:    string(invitation.Status),
		ExpiresAt: invitation.ExpiresAt.Format(time.RFC3339), // Use standard format
	}

	// 11. Return success
	resp = &types.InvitationResponse{
		Success:    true,
		Message:    "Invitation sent successfully",
		Invitation: respInvitation,
	}
	return resp, nil
}
