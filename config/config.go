package config

import (
	// We might need an import for the module interface later,
	// but for now, a string slice for names is sufficient.

	"github.com/templwind/soul/db"
	"github.com/templwind/soul/ratelimiter"
	"github.com/templwind/soul/webserver"
)

// Config defines the configuration required by the Soul module.
// The consuming application is responsible for loading its own configuration
// (e.g., from YAML, env vars) and populating this struct.
type Config struct {
	webserver.WebServerConf                                          // Removed: Web server config (Host, Port, etc.) is managed by the consuming application.
	db.DBConfig                                                      // Database configuration (DSN, etc.)
	DBMaxIdleConns          int                                      `yaml:"DBMaxIdleConns,omitempty"`    // DB Pool: Max idle connections
	DBMaxOpenConns          int                                      `yaml:"DBMaxOpenConns,omitempty"`    // DB Pool: Max open connections
	DBConnMaxLifetime       int                                      `yaml:"DBConnMaxLifetime,omitempty"` // DB Pool: Max connection lifetime (seconds)
	Nats                    NatsConfig                               // NATS connection details
	Redis                   RedisConfig                              // Redis connection details
	Environment             string                                   // e.g., "development", "production"
	InternalAPISecret       string                                   `yaml:"InternalAPISecret,omitempty"` // Secret for internal API calls if needed
	Auth                    AuthConfig                               // Authentication settings (JWT secrets, cookie names, OAuth)
	RateLimiters            map[string]ratelimiter.RateLimiterConfig // Rate limiter configurations
	GPT                     GPTConfig                                `yaml:"GPT,omitempty"`          // OpenAI GPT settings
	Anthropic               AnthropicConfig                          `yaml:"Anthropic,omitempty"`    // Anthropic settings
	AWS                     AWSConfig                                `yaml:"AWS,omitempty"`          // AWS settings (e.g., for S3, SES)
	DigitalOcean            DigitalOceanConfig                       `yaml:"DigitalOcean,omitempty"` // DigitalOcean settings (e.g., Spaces)
	Stripe                  StripeConfig                             `yaml:"Stripe,omitempty"`       // Stripe settings (API keys, webhook secret)
	Email                   EmailConfig                              `yaml:"Email,omitempty"`        // Email sending configuration
	Admin                   AdminConfig                              `yaml:"Admin,omitempty"`        // Admin-specific settings (e.g., authorized domains)
	FrontendURL             string                                   `yaml:"FrontendURL,omitempty"`  // Base URL for the frontend (used in emails, redirects)

	// EnabledModules specifies which optional modules should be initialized.
	// The consuming application provides the names of the modules it wants to use.
	EnabledModules []string `yaml:"EnabledModules,omitempty"`

	// Removed: EmbeddedFS map[string]*embed.FS - Filesystem embedding is app-specific.
	// Removed: TotalInstances int - Instance count is an infrastructure/deployment concern.
}

// AuthConfig holds authentication related settings
type AuthConfig struct {
	AccessSecret            string
	AccessExpire            int64
	AccountCookieName       string
	UserCookieName          string
	SessionCookieName       string
	SecretKey               string // General secret key (e.g., for session encryption)
	GoogleOAuthStateString  string `yaml:"GoogleOAuthStateString,omitempty"`
	GoogleOAuthClientID     string `yaml:"GoogleOAuthClientID,omitempty"`
	GoogleOAuthClientSecret string `yaml:"GoogleOAuthClientSecret,omitempty"`
	GoogleOAuthRedirectURL  string `yaml:"GoogleOAuthRedirectURL,omitempty"`
}

// NatsConfig holds NATS connection details
type NatsConfig struct {
	URL string
}

// RedisConfig holds Redis connection details
type RedisConfig struct {
	URL string
}

// GPTConfig holds OpenAI GPT settings
type GPTConfig struct {
	Endpoint       string
	APIKey         string
	OrgID          string
	Model          string
	DallEModel     string `yaml:"DallEModel,omitempty"`
	DallEEndpoint  string `yaml:"DallEEndpoint,omitempty"`
	TotalRPM       int
	MaxConcurrency int
}

// AnthropicConfig holds Anthropic settings
type AnthropicConfig struct {
	APIKey         string
	Model          string
	Endpoint       string
	RequestsPerMin int
}

// StripeConfig holds Stripe settings
type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	PriceIDs       StripePriceIDs `yaml:"PriceIDs,omitempty"`
}

// StripePriceIDs holds the specific Stripe Price IDs for different billable items.
type StripePriceIDs struct {
	ReservedDomain string `yaml:"ReservedDomain,omitempty"`
	CustomDomain   string `yaml:"CustomDomain,omitempty"`
	// Add other price IDs as needed
}

// EmailConfig holds email sending configuration
type EmailConfig struct {
	From             string // Sender's email address
	ReplyTo          string // Address to receive replies, optional but recommended
	BaseURL          string // Base URL for links (used in email templates)
	UnsubscribeURL   string // URL to handle unsubscriptions
	UnsubscribeText  string // Text for unsubscribe link
	ListUnsubscribe  string // List-Unsubscribe header
	PrivacyPolicyURL string // Link to your privacy policy
	CompanyInfo      CompanyInfoConfig
}

// CompanyInfoConfig holds company details for emails
type CompanyInfoConfig struct {
	Name         string // Company name
	Address      string // Company address
	Phone        string // Company phone number
	SupportEmail string // Support email address
}

// AWSConfig holds AWS settings
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string // e.g., for S3
}

// DigitalOceanConfig holds DigitalOcean settings
type DigitalOceanConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string // e.g., for Spaces
	Endpoint        string
}

// AdminConfig holds admin-specific settings
type AdminConfig struct {
	AuthorizedDomains []string // Domains allowed for admin access
}
