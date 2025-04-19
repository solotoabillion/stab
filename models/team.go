package models

import (
	"fmt" // Import fmt for error wrapping
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Team represents a group of users collaborating.
type Team struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name    string    `gorm:"size:100;not null"`
	OwnerID uuid.UUID `gorm:"type:uuid;not null;index"` // Foreign key to the User who owns the team

	// --- Relationships ---
	Owner User `gorm:"foreignKey:OwnerID"` // Belongs To User (Owner)
	// Memberships []Membership `gorm:"foreignKey:TeamID"` // Has Many Memberships - Uncomment when Membership model is defined
	// Invitations []Invitation `gorm:"foreignKey:TeamID"` // Has Many Invitations - Uncomment when Invitation model is defined
}

// BeforeCreate hook to generate UUID if not set
func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	// Add other validation or default setting logic here if needed
	return nil
}

// FindTeamByID retrieves a team by its primary key (UUID).
func FindTeamByID(db *gorm.DB, teamID uuid.UUID) (*Team, error) {
	var team Team
	result := db.First(&team, teamID) // Find by primary key
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, result.Error
	}
	return &team, nil
}

// CreateTeamWithOwner creates a new team and its owner membership within a transaction.
func CreateTeamWithOwner(db *gorm.DB, teamName string, ownerID uuid.UUID) (*Team, error) {
	var newTeam Team
	err := db.Transaction(func(tx *gorm.DB) error {
		// Create the Team
		team := Team{
			Name:    teamName,
			OwnerID: ownerID,
		}
		// BeforeCreate hook will generate UUID
		if err := tx.Create(&team).Error; err != nil {
			// Consider logging here or letting the caller log
			return fmt.Errorf("failed to create team: %w", err)
		}
		newTeam = team // Store the created team

		// Create the Owner Membership
		membership := Membership{
			UserID: ownerID,
			TeamID: team.ID, // Use the ID of the just-created team
			Role:   RoleOwner,
		}
		// BeforeCreate hook will generate UUID and validate role
		if err := tx.Create(&membership).Error; err != nil {
			// Consider logging here or letting the caller log
			return fmt.Errorf("failed to create owner membership: %w", err)
		}

		// Transaction successful
		return nil
	})

	if err != nil {
		return nil, err // Return the error from the transaction
	}

	// Return the created team (without membership details, caller can fetch if needed)
	return &newTeam, nil
}

// CountTeams counts the total number of teams.
func CountTeams(db *gorm.DB) (int64, error) {
	var count int64
	result := db.Model(&Team{}).Count(&count)
	return count, result.Error
}
