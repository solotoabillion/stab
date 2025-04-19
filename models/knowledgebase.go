package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// KnowledgeBaseArticle represents an article in the internal/external knowledge base.
type KnowledgeBaseArticle struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Title     string         `gorm:"size:255;not null;index"`
	Body      string         `gorm:"type:text;not null"`
	Tags      datatypes.JSON `gorm:"type:jsonb"`     // Store tags as JSON array
	Source    string         `gorm:"size:100;index"` // e.g., 'internal', 'external', 'faq', etc.
	IsActive  bool           `gorm:"default:true;index"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook to generate UUID if not set
func (a *KnowledgeBaseArticle) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// CreateKnowledgeBaseArticle creates a new KB article.
func CreateKnowledgeBaseArticle(db *gorm.DB, article *KnowledgeBaseArticle) error {
	return db.Create(article).Error
}

// SearchKnowledgeBaseArticles searches for articles by query string in title, body, or tags (case-insensitive, only active).
func SearchKnowledgeBaseArticles(db *gorm.DB, query string) ([]KnowledgeBaseArticle, error) {
	var articles []KnowledgeBaseArticle
	q := "%" + query + "%"
	err := db.Where("is_active = ? AND (title ILIKE ? OR body ILIKE ? OR tags::text ILIKE ?)", true, q, q, q).Find(&articles).Error
	return articles, err
}

// SoftDeleteKnowledgeBaseArticle sets IsActive=false for the given article (soft delete).
func SoftDeleteKnowledgeBaseArticle(db *gorm.DB, id uuid.UUID) error {
	result := db.Model(&KnowledgeBaseArticle{}).Where("id = ? AND is_active = ?", id, true).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
