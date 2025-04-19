package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InvitationStatus defines the possible states of an invitation.
type InvitationStatus string

const (
	StatusPending  InvitationStatus = "pending"
	StatusAccepted InvitationStatus = "accepted"
	StatusDeclined InvitationStatus = "declined"
	StatusExpired  InvitationStatus = "expired"
)

// Invitation represents a request for a user (identified by email) to join a team.
type Invitation struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Email     string           `gorm:"size:255;not null;index"`                   // Email of the invited user
	TeamID    uuid.UUID        `gorm:"type:uuid;not null;index"`                  // Team they are invited to
	InviterID uuid.UUID        `gorm:"type:uuid;not null;index"`                  // User who sent the invitation
	Role      Role             `gorm:"type:varchar(20);not null"`                 // Role offered (cannot invite owners)
	Token     string           `gorm:"size:64;uniqueIndex;not null"`              // Secure, unique token for the invitation link
	Status    InvitationStatus `gorm:"type:varchar(20);not null;default:pending"` // Current status of the invitation
	ExpiresAt time.Time        `gorm:"not null"`                                  // When the invitation expires

	// --- Relationships ---
	Team    Team `gorm:"foreignKey:TeamID"`    // Belongs To Team
	Inviter User `gorm:"foreignKey:InviterID"` // Belongs To User (Inviter)
}

// GenerateInvitationToken creates a secure random token.
// Moved out of BeforeCreate for potential reuse, though could stay inline.
func GenerateInvitationToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BeforeCreate hook to generate a unique token, set expiry, default status, and validate role.
func (inv *Invitation) BeforeCreate(tx *gorm.DB) (err error) {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}

	// Generate secure random token if not already set (e.g., for testing)
	if inv.Token == "" {
		token, err := GenerateInvitationToken()
		if err != nil {
			return err // Propagate error
		}
		inv.Token = token
	}

	// Set default expiry (e.g., 7 days from now) if not set
	if inv.ExpiresAt.IsZero() {
		inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	// Set default status if not set
	if inv.Status == "" {
		inv.Status = StatusPending
	}

	// Ensure invited role is not owner
	if inv.Role == RoleOwner {
		return errors.New("cannot invite user as owner")
	}
	// Ensure role is valid
	switch inv.Role {
	case RoleAdmin, RoleMember:
		// Valid roles for invitation
	default:
		return errors.New("invalid role for invitation")
	}

	return nil
}

// FindInvitationByTokenWithTeam retrieves an invitation by its unique token, preloading the associated Team.
func FindInvitationByTokenWithTeam(db *gorm.DB, token string) (*Invitation, error) {
	var invitation Invitation
	result := db.Preload("Team").Where("token = ?", token).First(&invitation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, result.Error
	}
	return &invitation, nil
}

// FindPendingInvitationsByTeam retrieves all pending invitations for a specific team.
func FindPendingInvitationsByTeam(db *gorm.DB, teamID uuid.UUID) ([]Invitation, error) {
	var invitations []Invitation
	result := db.Where("team_id = ? AND status = ?", teamID, StatusPending).Find(&invitations)
	if result.Error != nil {
		return nil, result.Error
	}
	// Ensure empty slice instead of null if no results
	if invitations == nil {
		invitations = []Invitation{}
	}
	return invitations, nil
}

// FindPendingInvitationByTeamAndEmail retrieves a pending invitation for a specific email and team.
func FindPendingInvitationByTeamAndEmail(db *gorm.DB, teamID uuid.UUID, email string) (*Invitation, error) {
	var invitation Invitation
	result := db.Where("team_id = ? AND email = ? AND status = ?", teamID, email, StatusPending).First(&invitation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, result.Error
	}
	return &invitation, nil
}

// FindInvitationByIDAndTeam retrieves a specific invitation by its ID, ensuring it belongs to the specified team.
func FindInvitationByIDAndTeam(db *gorm.DB, invitationID uuid.UUID, teamID uuid.UUID) (*Invitation, error) {
	var invitation Invitation
	result := db.Where("id = ? AND team_id = ?", invitationID, teamID).First(&invitation)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error like "invitation not found for this team"
		}
		return nil, result.Error
	}
	return &invitation, nil
}

// CreateInvitation creates a new invitation record in the database.
// It relies on the BeforeCreate hook to set defaults (ID, Token, ExpiresAt, Status) and validate the role.
func CreateInvitation(db *gorm.DB, invitation *Invitation) error {
	result := db.Create(invitation)
	return result.Error
}

// AcceptInvitation handles the process of a user accepting an invitation.
// It finds the invitation, checks its validity, creates/finds the membership,
// and updates the invitation status within a transaction.
// Returns the created/found membership record and the final invitation record, or an error.
func AcceptInvitation(db *gorm.DB, userID uuid.UUID, token string) (*Membership, *Invitation, error) { // Modified return type
	var invitation Invitation
	var membership Membership

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Find the invitation by token
		invResult := tx.Where("token = ?", token).First(&invitation)
		if invResult.Error != nil {
			if invResult.Error == gorm.ErrRecordNotFound {
				return gorm.ErrRecordNotFound // Specific error for not found
			}
			return invResult.Error // Other DB error
		}

		// 2. Check invitation status
		if invitation.Status != StatusPending {
			return errors.New("invitation is no longer pending") // Custom error
		}

		// 3. Check expiry
		if time.Now().After(invitation.ExpiresAt) {
			// Optionally update status to Expired
			// tx.Model(&invitation).Update("status", StatusExpired)
			return errors.New("invitation has expired") // Custom error
		}

		// 4. Check if user accepting matches the invited email (important if user logged in with different email)
		var user User
		userResult := tx.First(&user, userID)
		if userResult.Error != nil {
			// This shouldn't happen if the userID came from valid auth context
			return errors.New("failed to find accepting user")
		}
		if user.Email != invitation.Email {
			// Log this potential issue
			return errors.New("authenticated user email does not match invited email")
		}

		// 5. Create or find Membership record
		membership = Membership{
			UserID: userID,
			TeamID: invitation.TeamID,
			Role:   invitation.Role, // Assign role from invitation
		}
		// Use FirstOrCreate to handle potential existing membership (though unlikely if invite was pending)
		memResult := tx.Where("user_id = ? AND team_id = ?", userID, invitation.TeamID).FirstOrCreate(&membership)
		if memResult.Error != nil {
			return memResult.Error // Error creating/finding membership
		}
		// If FirstOrCreate found an existing record, update the role if different?
		// Current logic assumes a new membership or existing one is fine.
		// If role update is needed: if memResult.RowsAffected == 0 && membership.Role != invitation.Role { ... update role ... }

		// 6. Update Invitation status to Accepted
		updateResult := tx.Model(&invitation).Where("status = ?", StatusPending).Update("status", StatusAccepted)
		if updateResult.Error != nil {
			return updateResult.Error // Error updating invitation status
		}
		if updateResult.RowsAffected == 0 {
			// Status might have changed between check and update
			// Re-fetch the invitation to get the current status before returning the error
			tx.Where("token = ?", token).First(&invitation)
			return errors.New("failed to update invitation status, it might have been accepted or declined already")
		}
		// Update the local invitation struct status after successful update
		invitation.Status = StatusAccepted

		// Transaction successful
		return nil
	})

	if err != nil {
		// Even if there's an error (like race condition), return the invitation state found during the transaction
		return nil, &invitation, err // Modified return
	}

	// Return the created/found membership and the updated invitation
	return &membership, &invitation, nil // Modified return
}

// DeleteInvitation performs a soft delete on an invitation record by its ID.
func DeleteInvitation(db *gorm.DB, invitationID uuid.UUID) error {
	// Find the invitation first to ensure it exists before deleting
	var invitation Invitation
	findResult := db.First(&invitation, invitationID) // Find by primary key
	if findResult.Error != nil {
		if errors.Is(findResult.Error, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound // Or a custom error
		}
		return findResult.Error // Other DB error during find
	}

	// Perform the soft delete using the found object
	deleteResult := db.Delete(&invitation)
	if deleteResult.Error != nil {
		return deleteResult.Error
	}
	if deleteResult.RowsAffected == 0 {
		// This might happen in a race condition
		return gorm.ErrRecordNotFound // Or a custom error indicating it was already deleted/not found
	}
	return nil
}

// UpdateInvitationStatus updates the status of a specific invitation, ensuring it's currently pending.
// Returns the updated invitation or an error.
func UpdateInvitationStatus(db *gorm.DB, invitationID uuid.UUID, newStatus InvitationStatus) (*Invitation, error) {
	var invitation Invitation
	// Find the invitation first to ensure it exists
	findResult := db.First(&invitation, invitationID)
	if findResult.Error != nil {
		if errors.Is(findResult.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, findResult.Error // Other DB error during find
	}

	// Check if it's already the target status
	if invitation.Status == newStatus {
		return &invitation, nil // Already in the desired state
	}

	// Only allow updating from Pending status (except maybe to Expired, handled elsewhere)
	if invitation.Status != StatusPending {
		return &invitation, errors.New("invitation is no longer pending")
	}

	// Perform the update, ensuring the status is still Pending (prevents race conditions)
	updateResult := db.Model(&invitation).
		Where("status = ?", StatusPending).
		Update("status", newStatus)

	if updateResult.Error != nil {
		return nil, updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		// Status changed between find and update
		// Re-fetch to return the current state with the error
		db.First(&invitation, invitationID)
		return &invitation, errors.New("invitation status changed unexpectedly")
	}

	// Update the local struct status and return
	invitation.Status = newStatus
	return &invitation, nil
}
