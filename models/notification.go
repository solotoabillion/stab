package models

import (
	"errors" // Added
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification represents a message or alert for a user.
type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index"` // Added index for sorting
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	UserID uuid.UUID  `gorm:"type:uuid;index;not null"` // The user this notification is for
	Type   string     `gorm:"size:50;index"`            // Category/type (e.g., 'billing', 'security', 'team', 'general')
	Title  string     `gorm:"size:255;not null"`
	Body   string     `gorm:"type:text"`           // Longer description or details
	IsRead bool       `gorm:"default:false;index"` // Whether the user has read the notification
	ReadAt *time.Time `gorm:"index"`               // Timestamp when it was marked as read

	// Optional: Link to related resource (e.g., team ID, invoice ID)
	// RelatedResourceType string `gorm:"size:50;index"`
	// RelatedResourceID   string `gorm:"size:100;index"` // Use string for flexibility (UUIDs, IDs, etc.)

	// --- Relationships ---
	User User `gorm:"foreignKey:UserID"` // Belongs To User
}

// BeforeCreate hook to generate UUID if not set
func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// CreateNotification creates a new notification record in the database.
// It relies on the BeforeCreate hook to set the ID.
func CreateNotification(db *gorm.DB, notification *Notification) error {
	if notification.UserID == uuid.Nil {
		return errors.New("UserID is required to create a notification")
	}
	if notification.Title == "" {
		return errors.New("Title is required to create a notification")
	}
	// Add other validation as needed (e.g., Type)

	result := db.Create(notification)
	return result.Error
}

// FindNotificationsByUserID retrieves all notifications for a specific user, ordered by creation date descending.
func FindNotificationsByUserID(db *gorm.DB, userID uuid.UUID) ([]Notification, error) {
	var notifications []Notification
	result := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications)
	if result.Error != nil {
		return nil, result.Error
	}
	// Ensure empty slice instead of null if no results
	if notifications == nil {
		notifications = []Notification{}
	}
	return notifications, nil
}

// FindNotificationByIDAndUser retrieves a specific notification by its ID and user ID.
func FindNotificationByIDAndUser(db *gorm.DB, userID uuid.UUID, notificationID uuid.UUID) (*Notification, error) {
	var notification Notification
	result := db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification)
	if result.Error != nil {
		// Return specific error for not found, otherwise the GORM error
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // Or a custom error
		}
		return nil, result.Error
	}
	return &notification, nil
}

// MarkNotificationAsRead marks a specific notification as read for a user.
// It handles the transaction and checks if the notification exists and belongs to the user.
// Returns the updated notification or an error (gorm.ErrRecordNotFound if not found/not owned, or other DB errors).
// Note: It does NOT return an error if the notification was already read (idempotent).
func MarkNotificationAsRead(db *gorm.DB, userID uuid.UUID, notificationID uuid.UUID) (*Notification, error) {
	var notification Notification
	now := time.Now()

	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error // Failed to begin transaction
	}

	// Find the specific notification for the user that is not read
	result := tx.Model(&Notification{}).
		Where("id = ? AND user_id = ? AND is_read = ?", notificationID, userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error // Error during update
	}

	if result.RowsAffected == 0 {
		// Check if it exists but was already read or doesn't belong to user
		errCheck := tx.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error
		if errCheck == gorm.ErrRecordNotFound {
			tx.Rollback()
			return nil, gorm.ErrRecordNotFound // Not found or doesn't belong to user
		} else if errCheck == nil && notification.IsRead {
			// Already read, commit transaction and return the existing notification
			if errCommit := tx.Commit().Error; errCommit != nil {
				// Log error but proceed, as the state is effectively correct
				// Consider how to handle logging here if needed
			}
			return &notification, nil // Success (idempotent)
		} else if errCheck != nil {
			// Other error during check
			tx.Rollback()
			return nil, errCheck
		}
		// RowsAffected was 0 for another reason (shouldn't happen with First check)
		tx.Rollback()
		// Consider returning a generic error here
		return nil, gorm.ErrInvalidData // Or a more specific error
	}

	// Fetch the updated notification details after successful update
	if errFetch := tx.Where("id = ?", notificationID).First(&notification).Error; errFetch != nil {
		tx.Rollback()
		return nil, errFetch // Failed to fetch updated record
	}

	// Commit the transaction
	if errCommit := tx.Commit().Error; errCommit != nil {
		return nil, errCommit // Failed to commit
	}

	return &notification, nil // Success
}

// MarkAllNotificationsAsRead marks all unread notifications for a user as read.
// Returns the number of rows affected and any error.
func MarkAllNotificationsAsRead(db *gorm.DB, userID uuid.UUID) (rowsAffected int64, err error) {
	now := time.Now()
	result := db.Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	return result.RowsAffected, result.Error
}
