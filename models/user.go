package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors" // Import errors
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes" // Added for JSON type
	"gorm.io/gorm"
)

// SystemRole defines global roles within the application.
type SystemRole string

const (
	SystemRoleAdmin SystemRole = "admin" // Can access admin interface
	SystemRoleUser  SystemRole = "user"  // Standard user
)

// AccountStatus defines the possible states of a user account.
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	// Add other statuses like "pending_verification", "deactivated" if needed
)

// User represents a user in the system mapped to the database schema.
type User struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email                  string         `gorm:"uniqueIndex;not null;size:255"` // Added size limit
	Password               *string        `gorm:""`                              // Nullable for OAuth users
	CreatedAt              time.Time      `gorm:"autoCreateTime"`                // GORM handles this
	UpdatedAt              time.Time      `gorm:"autoUpdateTime"`                // GORM handles this
	DeletedAt              gorm.DeletedAt `gorm:"index"`                         // Support soft deletes
	ApiKey                 string         `gorm:"uniqueIndex;size:64"`
	PasswordResetToken     *string        `gorm:"index;size:64"`                   // Nullable password reset token
	PasswordResetExpiresAt *time.Time     ``                                       // Nullable expiry time for the token
	Plan                   string         `gorm:"not null;default:'free';size:50"` // Increased size slightly
	StripeCustomerID       *string        `gorm:"size:100;uniqueIndex"`            // Nullable Stripe Customer ID
	DefaultSubdomain       string         `gorm:"uniqueIndex;size:100"`
	ProfileData            datatypes.JSON `gorm:""`                                           // JSON blob for profile/OAuth data (first name, last name, avatar, provider, etc.)
	Role                   SystemRole     `gorm:"type:varchar(20);not null;default:'user'"`   // User's global system role
	AccountStatus          AccountStatus  `gorm:"type:varchar(20);not null;default:'active'"` // User account status
	Settings               datatypes.JSON `gorm:"type:jsonb"`

	// --- Associations ---
	// Define associations here if needed, e.g.:
	Subscriptions []Subscription `gorm:"foreignKey:UserID"` // Added for preloading
	Memberships   []Membership   `gorm:"foreignKey:UserID"` // Added for preloading
	// Teams         []Team         `gorm:"many2many:team_memberships;"` // Example many2many

	// Add other fields as needed: IsVerified, LastLoginAt, etc.
}

// UserSettings holds user preferences and feature flags.
type UserSettings struct {
	TwoFactorEnabled       bool `json:"twoFactorEnabled"`
	MarketingEmailsEnabled bool `json:"marketingEmailsEnabled"`
	SecurityAlertsEnabled  bool `json:"securityAlertsEnabled"`
	// Add more settings as needed
}

// GetSettings unmarshals the Settings JSON field into a UserSettings struct.
func (u *User) GetSettings() (UserSettings, error) {
	var s UserSettings
	if len(u.Settings) == 0 {
		return s, nil // Return zero value if not set
	}
	if err := json.Unmarshal(u.Settings, &s); err != nil {
		return s, err
	}
	return s, nil
}

// UpdateSettings marshals the given UserSettings and updates the Settings field.
func (u *User) UpdateSettings(db *gorm.DB, newSettings UserSettings) error {
	data, err := json.Marshal(newSettings)
	if err != nil {
		return err
	}
	u.Settings = datatypes.JSON(data)
	return db.Model(u).Update("settings", u.Settings).Error
}

// SetPassword hashes the given password and sets it on the user model.
// If the password string is empty, it sets the user's password to nil.
func (u *User) SetPassword(password string) error {
	if password == "" {
		u.Password = nil // Allow clearing password for OAuth users perhaps
		return nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashedStr := string(hashedPassword)
	u.Password = &hashedStr
	return nil
}

// CheckPassword compares the given password with the hashed password stored for the user.
// Returns false if the user has no password set (e.g., OAuth user).
func (u *User) CheckPassword(password string) bool {
	if u.Password == nil {
		return false // No password set
	}
	err := bcrypt.CompareHashAndPassword([]byte(*u.Password), []byte(password))
	return err == nil
}

// FindUserByID retrieves a user by their UUID from the database.
func FindUserByID(db *gorm.DB, id uuid.UUID) (*User, error) {
	var user User
	err := db.Where("id = ?", id).First(&user).Error
	if err != nil {
		// Consider returning specific errors like gorm.ErrRecordNotFound
		return nil, err
	}
	return &user, nil
}

// FindUserByEmail retrieves a user by their email address.
func FindUserByEmail(db *gorm.DB, email string) (*User, error) {
	var user User
	err := db.Where("email = ?", email).First(&user).Error
	if err != nil {
		// Return specific error for not found, otherwise the GORM error
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, err
	}
	return &user, nil
}

// GenerateAPIKey creates a secure random API key (e.g., 64 hex characters)
// Note: This might be better placed in a utility/service package if used elsewhere.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BeforeCreate hook to generate API key if not present
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New() // Generate UUID if not set
	}
	if u.ApiKey == "" {
		apiKey, err := GenerateAPIKey()
		if err != nil {
			return err
		}
		u.ApiKey = apiKey
	}
	// You might want to generate DefaultSubdomain here too if it's meant to be random
	return nil
}

// UpdateUserPasswordResetToken sets the password reset token and expiry for a user.
func UpdateUserPasswordResetToken(db *gorm.DB, userID uuid.UUID, token string, expiresAt time.Time) error {
	result := db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_reset_token":      &token,
		"password_reset_expires_at": &expiresAt,
	})
	return result.Error
}

// CreateUser creates a new user record in the database.
// It relies on the BeforeCreate hook to set defaults (ID, ApiKey).
// It assumes the password has already been hashed using user.SetPassword().
func CreateUser(db *gorm.DB, user *User) error {
	result := db.Create(user)
	return result.Error
}

// UpdateUserAPIKey updates the API key for a specific user.
func UpdateUserAPIKey(db *gorm.DB, userID uuid.UUID, newAPIKey string) error {
	result := db.Model(&User{}).Where("id = ?", userID).Update("api_key", newAPIKey)
	return result.Error
}

// FindUserByValidPasswordResetToken retrieves a user by a non-expired password reset token.
func FindUserByValidPasswordResetToken(db *gorm.DB, token string) (*User, error) {
	var user User
	err := db.Where("password_reset_token = ? AND password_reset_expires_at > ?", token, time.Now()).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error for invalid/expired token
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUserPasswordAndClearToken updates the user's password hash (making it nullable) and clears the reset token fields.
func UpdateUserPasswordAndClearToken(db *gorm.DB, userID uuid.UUID, newHashedPassword *string) error {
	// Use map to explicitly set token fields to NULL and handle nullable password
	updates := map[string]interface{}{
		"password":                  newHashedPassword, // Can be nil
		"password_reset_token":      nil,
		"password_reset_expires_at": nil,
	}
	result := db.Model(&User{}).Where("id = ?", userID).Updates(updates)
	return result.Error
}

// FindAllUsersForAdmin retrieves all users, selecting specific fields suitable for admin listing.
// TODO: Add pagination, filtering, sorting parameters.
func FindAllUsersForAdmin(db *gorm.DB) ([]User, error) {
	var users []User
	// Select specific fields to avoid exposing sensitive data like password hash
	// Adjust fields based on what the admin list actually needs
	result := db.Select("id", "email", "profile_data", "role", "account_status", "created_at", "updated_at", "api_key", "default_subdomain", "plan", "stripe_customer_id").
		Order("created_at asc"). // Order by creation time, or ID, or email?
		Find(&users)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error // Return actual error, allow ErrRecordNotFound
	}

	// Ensure empty slice instead of nil if no results found
	if users == nil {
		users = []User{}
	}

	return users, nil
}

// FindUserWithDetailsForAdmin retrieves a user by ID with preloaded associations needed for admin view.
func FindUserWithDetailsForAdmin(db *gorm.DB, userID uuid.UUID) (*User, error) {
	var user User
	result := db.
		Preload("Subscriptions.Plan"). // Load subscriptions and their plans
		Preload("Memberships.Team").   // Load memberships and their teams
		Where("id = ?", userID).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // Explicitly return not found error
		}
		return nil, result.Error // Return other DB errors
	}
	return &user, nil
}

// CountNonAdminUsers counts the total number of users excluding admins.
func CountNonAdminUsers(db *gorm.DB) (int64, error) {
	var count int64
	result := db.Model(&User{}).Where("role <> ?", SystemRoleAdmin).Count(&count)
	return count, result.Error
}
