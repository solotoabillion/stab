package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Plan defines the structure for subscription plans.
type Plan struct {
	ID                  string         `gorm:"primaryKey;size:50"`
	Name                string         `gorm:"size:100;not null"`
	StripePriceID       string         `gorm:"size:100;index"` // Monthly price ID
	Features            datatypes.JSON `gorm:"type:jsonb"`     // Store features as JSON
	PriceMonthly        float64        `gorm:"type:decimal(10,2)"`
	StripePriceIDYearly *string        `gorm:"size:100;index"` // Optional yearly price ID
	PriceYearly         *float64       `gorm:"type:decimal(10,2)"`
	Active              bool           `gorm:"default:true;index"`
	CreatedAt           time.Time      `gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"` // Add soft delete support
}

// Subscription tracks user subscriptions.
type Subscription struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID               uuid.UUID `gorm:"type:uuid;not null;index"`
	PlanID               string    `gorm:"size:50;not null;index"` // Foreign key to Plan.ID
	StripeSubscriptionID string    `gorm:"size:100;uniqueIndex;not null"`
	Status               string    `gorm:"size:50;not null;index"` // e.g., active, past_due, canceled, trialing
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CancelAtPeriodEnd    bool      `gorm:"default:false"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
}

// SubscriptionItem represents an item (add-on) on a subscription.
type SubscriptionItem struct {
	ID                       uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	SubscriptionID           uuid.UUID  `gorm:"type:uuid;not null;index"`
	StripeSubscriptionItemID string     `gorm:"size:100;uniqueIndex;not null"`
	StripePriceID            string     `gorm:"size:100;not null"`
	ItemType                 string     `gorm:"size:50;not null;index"`
	RelatedResourceID        *uuid.UUID `gorm:"type:uuid;index"`
	Quantity                 int        `gorm:"default:1"`
	CreatedAt                time.Time  `gorm:"autoCreateTime"`
	UpdatedAt                time.Time  `gorm:"autoUpdateTime"`
}

func CreateSubscription(ctx context.Context, db *gorm.DB, userID string, planID string, stripeSubscriptionID string, status string) (*Subscription, error) {
	subscription := &Subscription{
		UserID:               uuid.MustParse(userID),
		PlanID:               planID,
		StripeSubscriptionID: stripeSubscriptionID,
		Status:               status,
	}
	if err := db.WithContext(ctx).Create(subscription).Error; err != nil {
		return nil, err
	}
	return subscription, nil
}

// FindPlanByID fetches a Plan by its string ID.
func FindPlanByID(db *gorm.DB, id string) (*Plan, error) {
	var plan Plan
	if err := db.Where("id = ?", id).First(&plan).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

// CountActiveSubscriptions returns the number of active subscriptions.
func CountActiveSubscriptions(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&Subscription{}).Where("status = ?", "active").Count(&count).Error
	return count, err
}

// FindActivePlans returns all active plans.
func FindActivePlans(db *gorm.DB) ([]Plan, error) {
	var plans []Plan
	err := db.Where("active = ?", true).Find(&plans).Error
	return plans, err
}

// UnmarshalJSONFeatures converts datatypes.JSON to a []string slice.
func UnmarshalJSONFeatures(features datatypes.JSON) []string {
	var out []string
	_ = json.Unmarshal(features, &out)
	return out
}

// MarshalJSONFeatures converts a []string slice to datatypes.JSON.
func MarshalJSONFeatures(features []string) datatypes.JSON {
	b, _ := json.Marshal(features)
	return datatypes.JSON(b)
}

// FindSubscriptionItemByTypeAndResource fetches a SubscriptionItem by subscription ID, item type, and related resource ID.
func FindSubscriptionItemByTypeAndResource(db *gorm.DB, subscriptionID uuid.UUID, itemType string, resourceID uuid.UUID) (*SubscriptionItem, error) {
	var item SubscriptionItem
	err := db.Where("subscription_id = ? AND item_type = ? AND related_resource_id = ?", subscriptionID, itemType, resourceID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateSubscriptionItem inserts a new SubscriptionItem into the database.
func CreateSubscriptionItem(db *gorm.DB, item *SubscriptionItem) error {
	return db.Create(item).Error
}

// FindLatestActiveSubscriptionByUserID fetches the most recent active subscription for a user.
func FindLatestActiveSubscriptionByUserID(db *gorm.DB, userID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := db.Where("user_id = ? AND status = ?", userID, "active").Order("created_at DESC").First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
