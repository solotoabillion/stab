package models

import (
	// Import errors package
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BlogPostStatus defines the possible states of a blog post.
type BlogPostStatus string

const (
	StatusDraft     BlogPostStatus = "draft"
	StatusPublished BlogPostStatus = "published"
	StatusArchived  BlogPostStatus = "archived"
)

// BlogPost represents an article or entry in the blog.
type BlogPost struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Title       string         `gorm:"size:255;not null;index"`
	Slug        string         `gorm:"size:255;not null;uniqueIndex"` // URL-friendly identifier
	Content     string         `gorm:"type:text"`                     // Can store Markdown or HTML
	Excerpt     string         `gorm:"type:text"`                     // Short summary
	AuthorID    *uuid.UUID     `gorm:"type:uuid;index"`               // Link to User model (nullable)
	Status      BlogPostStatus `gorm:"size:20;not null;default:draft;index"`
	IsFeatured  bool           `gorm:"default:false;index"` // For featuring posts
	CoverImage  string         `gorm:"size:512"`            // URL to cover image
	ReadTime    int            `gorm:"default:0"`           // Estimated read time in minutes
	ViewCount   int            `gorm:"default:0"`           // Number of views
	LikeCount   int            `gorm:"default:0"`           // Number of likes
	PublishedAt *time.Time     `gorm:"index"`               // Timestamp when the post was published

	// Optional fields
	FeaturedImageURL string `gorm:"size:512"`

	// --- Relationships ---
	Author     *User      `gorm:"foreignKey:AuthorID"` // Belongs To User (optional)
	Tags       []Tag      `gorm:"many2many:blog_post_tags;"`
	Categories []Category `gorm:"many2many:blog_post_categories;"`
}

// --- Slug Generation ---

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	dashRegex            = regexp.MustCompile(`-{2,}`)
)

// generateSlug creates a URL-friendly slug from a title.
func generateSlug(title string) string {
	lower := strings.ToLower(title)
	noSpecial := nonAlphanumericRegex.ReplaceAllString(lower, "-")
	noMultipleDashes := dashRegex.ReplaceAllString(noSpecial, "-")
	trimmed := strings.Trim(noMultipleDashes, "-")
	// Note: Uniqueness check ideally happens in the handler/logic layer before saving
	// to avoid complex DB lookups within the hook and handle potential race conditions.
	// If a collision occurs, the handler can append a suffix (e.g., "-1", "-2").
	return trimmed
}

// --- Hooks ---

// BeforeSave hook to automatically generate slug if empty and set PublishedAt.
func (p *BlogPost) BeforeSave(tx *gorm.DB) (err error) {
	// Generate slug from title if slug is empty and title is present
	if p.Slug == "" && p.Title != "" {
		p.Slug = generateSlug(p.Title)
		// Consider adding a check in the logic layer to ensure slug uniqueness before calling save.
	}

	// Manage PublishedAt timestamp based on Status changes
	// Check if Status field is actually being changed in this transaction
	if tx.Statement.Changed("Status") {
		if p.Status == StatusPublished {
			// If changing to Published and PublishedAt is not already set
			if p.PublishedAt == nil {
				now := time.Now().UTC() // Use UTC for consistency
				p.PublishedAt = &now
			}
		} else {
			// If changing to any status other than Published, clear PublishedAt
			p.PublishedAt = nil
		}
	} else if p.Status == StatusPublished && p.PublishedAt == nil {
		// Handle case where record is created directly as Published
		now := time.Now().UTC()
		p.PublishedAt = &now
	}

	// Generate UUID if not set (relevant for BeforeCreate)
	if p.ID == uuid.Nil && tx.Statement.Schema != nil && tx.Statement.Schema.PrioritizedPrimaryField != nil && tx.Statement.Schema.PrioritizedPrimaryField.Name == "ID" {
		// This check is more robust for hooks that might run in different contexts
		p.ID = uuid.New()
	}

	return nil
}

// --- Query Functions ---

// FindPublishedBlogPosts retrieves a paginated and sorted list of published blog posts.
func FindPublishedBlogPosts(db *gorm.DB, params BlogPostsQueryParams) ([]BlogPost, int64, error) {
	var posts []BlogPost
	var total int64

	query := db.Model(&BlogPost{}).
		Preload("Author").
		Preload("Tags").
		Preload("Categories")

	// Apply filters
	query = query.Where("status = ?", StatusPublished)
	if params.Featured {
		query = query.Where("is_featured = ?", true)
	}
	if params.AuthorID != "" {
		query = query.Where("author_id = ?", params.AuthorID)
	}
	if params.Search != "" {
		searchTerm := "%" + params.Search + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ? OR excerpt ILIKE ?", searchTerm, searchTerm, searchTerm)
	}

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	switch params.Sort {
	case "oldest":
		query = query.Order("published_at ASC")
	case "popular":
		query = query.Order("view_count DESC, published_at DESC")
	case "featured":
		query = query.Order("is_featured DESC, published_at DESC")
	default: // "newest" or any other value
		query = query.Order("published_at DESC")
	}

	// Apply pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	// Execute query
	if err := query.Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// BlogPostsQueryParams holds the parameters for querying blog posts
type BlogPostsQueryParams struct {
	Limit    int
	Offset   int
	Sort     string
	Featured bool
	AuthorID string
	Search   string
}

// CountBlogPosts counts the total number of blog posts (optionally filter by status).
func CountBlogPosts(db *gorm.DB, statusFilter ...BlogPostStatus) (int64, error) {
	var count int64
	query := db.Model(&BlogPost{})
	if len(statusFilter) > 0 {
		query = query.Where("status IN ?", statusFilter)
	}
	result := query.Count(&count)
	return count, result.Error
}

// IncrementViewCount atomically increments the view count of a blog post
func (p *BlogPost) IncrementViewCount(db *gorm.DB) error {
	return db.Model(p).Update("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementLikeCount atomically increments the like count of a blog post
func (p *BlogPost) IncrementLikeCount(db *gorm.DB) error {
	return db.Model(p).Update("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount atomically decrements the like count of a blog post
func (p *BlogPost) DecrementLikeCount(db *gorm.DB) error {
	return db.Model(p).Where("like_count > 0").Update("like_count", gorm.Expr("like_count - ?", 1)).Error
}

// CalculateReadTime estimates reading time based on content length
func (p *BlogPost) CalculateReadTime() {
	// Average reading speed (words per minute)
	const wordsPerMinute = 200

	// Count words in content (simple split by spaces)
	words := len(strings.Fields(p.Content))

	// Calculate minutes, round up to nearest minute
	minutes := (words + wordsPerMinute - 1) / wordsPerMinute
	if minutes < 1 {
		minutes = 1
	}

	p.ReadTime = minutes
}

// BeforeCreate hook to calculate read time before creating post
func (p *BlogPost) BeforeCreate(tx *gorm.DB) error {
	p.CalculateReadTime()
	return nil
}

// BeforeUpdate hook to recalculate read time when content changes
func (p *BlogPost) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Content") {
		p.CalculateReadTime()
	}
	return nil
}

// FindPublishedBlogPostBySlug retrieves a single published blog post by its slug.
func FindPublishedBlogPostBySlug(db *gorm.DB, slug string) (*BlogPost, error) {
	var post BlogPost
	result := db.Preload("Author").Preload("Tags").Preload("Categories").
		Where("slug = ? AND status = ?", slug, StatusPublished).
		First(&post)
	if result.Error != nil {
		return nil, result.Error
	}
	return &post, nil
}

// FindPublishedBlogPostsByTag retrieves published blog posts for a given tag ID, with pagination.
func FindPublishedBlogPostsByTag(db *gorm.DB, tagID string, limit, offset int) ([]BlogPost, int64, error) {
	var posts []BlogPost
	var total int64

	query := db.Model(&BlogPost{}).
		Joins("JOIN blog_post_tags ON blog_posts.id = blog_post_tags.blog_post_id").
		Where("blog_post_tags.tag_id = ? AND blog_posts.status = ?", tagID, StatusPublished).
		Preload("Author").Preload("Tags").Preload("Categories")

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	query = query.Order("blog_posts.published_at DESC")

	if err := query.Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// FindPublishedBlogPostsByCategory retrieves published blog posts for a given category ID, with pagination.
func FindPublishedBlogPostsByCategory(db *gorm.DB, categoryID string, limit, offset int) ([]BlogPost, int64, error) {
	var posts []BlogPost
	var total int64

	query := db.Model(&BlogPost{}).
		Joins("JOIN blog_post_categories ON blog_posts.id = blog_post_categories.blog_post_id").
		Where("blog_post_categories.category_id = ? AND blog_posts.status = ?", categoryID, StatusPublished).
		Preload("Author").Preload("Tags").Preload("Categories")

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	query = query.Order("blog_posts.published_at DESC")

	if err := query.Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}
