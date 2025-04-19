package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Setting represents a dynamic application setting stored in the database.
type Setting struct {
	Category    string    `gorm:"primaryKey;size:64" json:"category"`
	Key         string    `gorm:"primaryKey;size:128" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	DataType    string    `gorm:"size:32;not null;default:string" json:"dataType"` // string, int, bool, json, etc.
	SortBy      int       `gorm:"not null;default:0" json:"sortBy"`
	Visibility  string    `gorm:"size:16;not null;default:admin" json:"visibility"` // admin, client, both
	Description string    `gorm:"type:text" json:"description"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// LoadSettingsByCategory loads all settings for a given category as a map[key]value.
func LoadSettingsByCategory(db *gorm.DB, category string) (map[string]string, error) {
	var settings []Setting
	result := db.Where("category = ?", category).Find(&settings)
	if result.Error != nil {
		return nil, result.Error
	}
	settingsMap := make(map[string]string, len(settings))
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}
	return settingsMap, nil
}

// SettingsMap is a map of category -> key -> value for all settings.
type SettingsMap map[string]map[string]string

// LoadAllSettings loads all settings from the DB as a SettingsMap.
func LoadAllSettings(db *gorm.DB) (SettingsMap, error) {
	var settings []Setting
	result := db.Find(&settings)
	if result.Error != nil {
		return nil, result.Error
	}
	settingsMap := make(SettingsMap)
	for _, s := range settings {
		if _, ok := settingsMap[s.Category]; !ok {
			settingsMap[s.Category] = make(map[string]string)
		}
		settingsMap[s.Category][s.Key] = s.Value
	}
	return settingsMap, nil
}

// DefaultSettings defines the recommended keys for all major providers.
var DefaultSettings = []Setting{
	// AI LLMs
	{Category: "ai_llms/chatgpt", Key: "api_key", DataType: "string", Description: "OpenAI API key", Visibility: "admin"},
	{Category: "ai_llms/chatgpt", Key: "org_id", DataType: "string", Description: "OpenAI organization ID", Visibility: "admin"},
	{Category: "ai_llms/claudeai", Key: "api_key", DataType: "string", Description: "ClaudeAI API key", Visibility: "admin"},
	{Category: "ai_llms/gemini", Key: "api_key", DataType: "string", Description: "Gemini API key", Visibility: "admin"},
	// OAuth Providers
	{Category: "oauth_providers/google", Key: "client_id", DataType: "string", Description: "Google OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/google", Key: "client_secret", DataType: "string", Description: "Google OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/google", Key: "redirect_url", DataType: "string", Description: "Google OAuth Redirect URL", Visibility: "admin"},
	{Category: "oauth_providers/x", Key: "client_id", DataType: "string", Description: "X (Twitter) OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/x", Key: "client_secret", DataType: "string", Description: "X (Twitter) OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/x", Key: "redirect_url", DataType: "string", Description: "X (Twitter) OAuth Redirect URL", Visibility: "admin"},
	{Category: "oauth_providers/github", Key: "client_id", DataType: "string", Description: "GitHub OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/github", Key: "client_secret", DataType: "string", Description: "GitHub OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/github", Key: "redirect_url", DataType: "string", Description: "GitHub OAuth Redirect URL", Visibility: "admin"},
	{Category: "oauth_providers/meta", Key: "client_id", DataType: "string", Description: "Meta OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/meta", Key: "client_secret", DataType: "string", Description: "Meta OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/meta", Key: "redirect_url", DataType: "string", Description: "Meta OAuth Redirect URL", Visibility: "admin"},
	{Category: "oauth_providers/instagram", Key: "client_id", DataType: "string", Description: "Instagram OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/instagram", Key: "client_secret", DataType: "string", Description: "Instagram OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/instagram", Key: "redirect_url", DataType: "string", Description: "Instagram OAuth Redirect URL", Visibility: "admin"},
	{Category: "oauth_providers/apple", Key: "client_id", DataType: "string", Description: "Apple OAuth Client ID", Visibility: "admin"},
	{Category: "oauth_providers/apple", Key: "client_secret", DataType: "string", Description: "Apple OAuth Client Secret", Visibility: "admin"},
	{Category: "oauth_providers/apple", Key: "redirect_url", DataType: "string", Description: "Apple OAuth Redirect URL", Visibility: "admin"},
	// MCP Servers (example: default)
	{Category: "mcp/servers/default", Key: "endpoint", DataType: "string", Description: "Default MCP server endpoint URL", Visibility: "admin"},
	{Category: "mcp/servers/default", Key: "token", DataType: "string", Description: "Default MCP server auth token", Visibility: "admin"},
	{Category: "mcp/servers/default", Key: "description", DataType: "string", Description: "Description for this MCP server", Visibility: "admin"},
	// MCP Clients (example: default)
	{Category: "mcp/clients/default", Key: "endpoint", DataType: "string", Description: "Default MCP client endpoint URL", Visibility: "admin"},
	{Category: "mcp/clients/default", Key: "token", DataType: "string", Description: "Default MCP client auth token", Visibility: "admin"},
	{Category: "mcp/clients/default", Key: "description", DataType: "string", Description: "Description for this MCP client", Visibility: "admin"},
	// Billing - Stripe
	{Category: "billing/stripe", Key: "secret_key", DataType: "string", Description: "Stripe Secret Key", Visibility: "admin"},
	{Category: "billing/stripe", Key: "publishable_key", DataType: "string", Description: "Stripe Publishable Key", Visibility: "admin"},
	{Category: "billing/stripe", Key: "webhook_secret", DataType: "string", Description: "Stripe Webhook Secret", Visibility: "admin"},
	{Category: "billing/stripe", Key: "price_id_reserved_domain", DataType: "string", Description: "Stripe Price ID for Reserved Domain", Visibility: "admin"},
	{Category: "billing/stripe", Key: "price_id_custom_domain", DataType: "string", Description: "Stripe Price ID for Custom Domain", Visibility: "admin"},
	// Billing - Paddle (future)
	{Category: "billing/paddle", Key: "vendor_id", DataType: "string", Description: "Paddle Vendor ID", Visibility: "admin"},
	{Category: "billing/paddle", Key: "api_key", DataType: "string", Description: "Paddle API Key", Visibility: "admin"},
	// Billing - PayPal (future)
	{Category: "billing/paypal", Key: "client_id", DataType: "string", Description: "PayPal Client ID", Visibility: "admin"},
	{Category: "billing/paypal", Key: "client_secret", DataType: "string", Description: "PayPal Client Secret", Visibility: "admin"},
	// Email
	{Category: "email/ses", Key: "from_address", DataType: "string", Description: "Sender email address", Visibility: "admin"},
	{Category: "email/ses", Key: "aws_region", DataType: "string", Description: "AWS SES region", Visibility: "admin"},
	{Category: "email/ses", Key: "aws_access_key_id", DataType: "string", Description: "AWS SES access key", Visibility: "admin"},
	{Category: "email/ses", Key: "aws_secret_access_key", DataType: "string", Description: "AWS SES secret key", Visibility: "admin"},
}

// SeedDefaultSettings inserts any missing default keys into the DB (does not overwrite values).
func SeedDefaultSettings(db *gorm.DB, defaults []Setting) error {
	for _, s := range defaults {
		var existing Setting
		err := db.Where("category = ? AND key = ?", s.Category, s.Key).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			s.Value = "" // or a safe default
			db.Create(&s)
		}
	}
	return nil
}

// CreateSetting inserts a new setting.
func CreateSetting(db *gorm.DB, s Setting) error {
	return db.Create(&s).Error
}

// DeleteSetting deletes a setting by category and key.
func DeleteSetting(db *gorm.DB, category, key string) error {
	return db.Where("category = ? AND key = ?", category, key).Delete(&Setting{}).Error
}

// GetSetting fetches a setting by category and key.
func GetSetting(db *gorm.DB, category, key string) (Setting, error) {
	var s Setting
	err := db.Where("category = ? AND key = ?", category, key).First(&s).Error
	return s, err
}

// FindSettingsByCategoryPrefix fetches all settings with a category prefix.
func FindSettingsByCategoryPrefix(db *gorm.DB, prefix string) ([]Setting, error) {
	var settings []Setting
	err := db.Where("category LIKE ?", prefix+"%").Find(&settings).Error
	return settings, err
}

// FindAllSettings fetches all settings.
func FindAllSettings(db *gorm.DB) ([]Setting, error) {
	var settings []Setting
	err := db.Find(&settings).Error
	return settings, err
}

// UpdateSetting updates a setting by category and key.
func UpdateSetting(db *gorm.DB, s Setting) error {
	return db.Model(&Setting{}).Where("category = ? AND key = ?", s.Category, s.Key).Updates(map[string]interface{}{
		"value":       s.Value,
		"data_type":   s.DataType,
		"sort_by":     s.SortBy,
		"visibility":  s.Visibility,
		"description": s.Description,
	}).Error
}

// SaveSettings upserts a batch of settings.
func SaveSettings(db *gorm.DB, settings []Setting) error {
	for _, s := range settings {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "category"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "data_type", "sort_by", "visibility", "description"}),
		}).Create(&s).Error
		if err != nil {
			return err
		}
	}
	return nil
}
