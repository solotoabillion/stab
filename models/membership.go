package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role defines the possible roles a user can have within a team.
type Role string

const (
	RoleOwner  Role = "owner"  // Can manage billing, team settings, and members
	RoleAdmin  Role = "admin"  // Can manage members and team settings (but not billing)
	RoleMember Role = "member" // Standard member access
)

// Membership links a User to a Team with a specific Role.
type Membership struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_team;not null"` // Part of composite unique index
	TeamID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_team;not null"` // Part of composite unique index
	Role   Role      `gorm:"type:varchar(20);not null"`

	// --- Relationships ---
	User User `gorm:"foreignKey:UserID"` // Belongs To User
	Team Team `gorm:"foreignKey:TeamID"` // Belongs To Team
}

// BeforeCreate hook to generate UUID if not set and validate Role
func (m *Membership) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	// Validate Role
	switch m.Role {
	case RoleOwner, RoleAdmin, RoleMember:
		// Valid role
	default:
		return errors.New("invalid membership role")
	}
	return nil
}

// FindMembershipsByTeam retrieves all membership records for a specific team.
// Consider adding preloading for User if needed: db.Preload("User").Where(...)
func FindMembershipsByTeam(db *gorm.DB, teamID uuid.UUID) ([]Membership, error) {
	var memberships []Membership
	result := db.Where("team_id = ?", teamID).Find(&memberships)
	if result.Error != nil {
		return nil, result.Error
	}
	// Ensure empty slice instead of null if no results
	if memberships == nil {
		memberships = []Membership{}
	}
	return memberships, nil
}

// FindMembershipByUserAndTeam retrieves a membership record for a specific user and team.
func FindMembershipByUserAndTeam(db *gorm.DB, userID uuid.UUID, teamID uuid.UUID) (*Membership, error) {
	var membership Membership
	result := db.Where("user_id = ? AND team_id = ?", userID, teamID).First(&membership)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, result.Error
	}
	return &membership, nil
}

// BeforeUpdate hook to validate Role on update
func (m *Membership) BeforeUpdate(tx *gorm.DB) (err error) {
	// Validate Role if it's being changed
	if tx.Statement.Changed("Role") {
		switch m.Role {
		case RoleOwner, RoleAdmin, RoleMember:
			// Valid role
		default:
			return errors.New("invalid membership role")
		}
	}
	return nil
}

// UpdateMembershipRole updates the role for a specific membership record.
// It relies on the BeforeUpdate hook to validate the new role.
func UpdateMembershipRole(db *gorm.DB, membershipID uuid.UUID, newRole Role) error {
	// Validate role before attempting update (though BeforeUpdate hook also does this)
	switch newRole {
	case RoleAdmin, RoleMember:
		// Valid roles to update to
	default:
		return errors.New("invalid target role for update")
	}

	result := db.Model(&Membership{}).Where("id = ?", membershipID).Update("role", newRole)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // Membership with that ID not found
	}
	return nil
}

// DeleteMembershipByUserAndTeam performs a soft delete on a membership record
// based on the user ID and team ID. It finds the record first.
func DeleteMembershipByUserAndTeam(db *gorm.DB, userID uuid.UUID, teamID uuid.UUID) error {
	// Find the membership first to ensure it exists
	var membership Membership
	findResult := db.Where("user_id = ? AND team_id = ?", userID, teamID).First(&membership)
	if findResult.Error != nil {
		if errors.Is(findResult.Error, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound // Or a custom "membership not found" error
		}
		return findResult.Error // Other DB error during find
	}

	// Perform the soft delete using the found membership object
	// GORM handles soft delete via DeletedAt field automatically
	deleteResult := db.Delete(&membership)
	if deleteResult.Error != nil {
		return deleteResult.Error
	}
	if deleteResult.RowsAffected == 0 {
		// This might happen in a race condition if deleted between find and delete
		return gorm.ErrRecordNotFound // Or a custom error indicating it was already deleted/not found
	}
	return nil
}
