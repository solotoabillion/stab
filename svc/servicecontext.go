package svc

import (
	"fmt"
	"log"
	"sync" // Added for ModuleServices mutex
	"time" // Needed for DB connection pool settings

	// Use new top-level package paths
	"github.com/solotoabillion/stab/config"
	"github.com/solotoabillion/stab/core/cache"
	"github.com/solotoabillion/stab/core/communication"
	"github.com/solotoabillion/stab/core/jobs"
	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/db"
	"github.com/solotoabillion/stab/models" // Added import for models

	"github.com/redis/go-redis/v9"
	"github.com/templwind/soul/events" // Assuming this is a shared library
	"github.com/templwind/soul/pubsub"
	"github.com/templwind/soul/ratelimiter"
	"github.com/templwind/soul/webserver/sse" // Assuming this is a shared library
	"gorm.io/driver/postgres"                 // Need GORM driver
	"gorm.io/gorm"

	// "github.com/labstack/echo/v4" // Removed: Middleware/Echo types don't belong here
	_ "github.com/lib/pq" // Keep for GORM Postgres driver side effects
)

// ServiceContext holds shared resources and configurations for the Soul module.
// It is initialized by NewServiceContext based on the provided config.Config.
// It also implements the modules.ModuleContext interface.
type ServiceContext struct {
	Config              *config.Config           // Pointer to the configuration provided by the consumer
	DB                  *gorm.DB                 // GORM database connection
	RedisClient         *redis.Client            // Redis client connection (can be nil if not configured)
	CommunicationClient communication.Service    // Communication service (e.g., SES/SNS) (can be nil)
	Session             *session.Session         // Session manager
	RateLimiter         *ratelimiter.RateLimiter // Rate limiter instance
	JobManager          *jobs.JobManager         // Background job manager
	PubSubBroker        pubsub.Broker            // Pub/Sub broker (e.g., NATS or NoOp)
	EventHub            *sse.EventHub            // Server-Sent Events hub
	// EmailSender         email.EmailSender     // Removed: Handled by consumer

	moduleServicesMu sync.RWMutex           // Mutex for concurrent access to ModuleServices
	ModuleServices   map[string]interface{} // Registry for services provided by enabled modules
	Settings         models.SettingsMap     // Changed type to match LoadAllSettings return type
	// Removed Middleware fields: CustomStatic, NoCache, AdminRequired
	// Removed Settings field: Rely on Config directly
}

// --- modules.ModuleContext Implementation ---

// GetDB returns the GORM database instance.
func (s *ServiceContext) GetDB() *gorm.DB {
	return s.DB
}

// GetRedisClient returns the Redis client instance (may be nil).
func (s *ServiceContext) GetRedisClient() *redis.Client {
	return s.RedisClient
}

// GetConfig returns the module configuration.
func (s *ServiceContext) GetConfig() *config.Config {
	return s.Config
}

// AddModuleService registers a service provided by a module.
func (s *ServiceContext) AddModuleService(key string, service interface{}) {
	s.moduleServicesMu.Lock()
	defer s.moduleServicesMu.Unlock()
	if s.ModuleServices == nil {
		s.ModuleServices = make(map[string]interface{})
	}
	s.ModuleServices[key] = service
}

// GetModuleService retrieves a service registered by another module.
func (s *ServiceContext) GetModuleService(key string) (interface{}, bool) {
	s.moduleServicesMu.RLock()
	defer s.moduleServicesMu.RUnlock()
	service, ok := s.ModuleServices[key]
	return service, ok
}

// --- End modules.ModuleContext Implementation ---

// initGormDB initializes the GORM database connection and configures the pool.
// Accepts the main config to access pool settings.
func initGormDB(c *config.Config) (*gorm.DB, error) {
	// Use DSN from the embedded soulDB.DBConfig within the main config
	gormDB, err := gorm.Open(postgres.Open(c.DBConfig.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database using GORM: %w", err)
	}

	// Configure connection pool
	sqlDB, err := gormDB.DB()
	if err != nil {
		// Attempt to close the initial GORM connection if we can't get the underlying sql.DB
		// Note: Directly closing gormDB might not be standard practice, but necessary here.
		// Consider logging this specific error scenario.
		log.Printf("WARN: Failed to get underlying sql.DB, attempting to close initial GORM connection.")
		// There isn't a direct gormDB.Close(), closing the underlying sql.DB is the way.
		// Since we failed to get it via gormDB.DB(), we might not be able to close it easily here.
		// Let's just return the error. The caller (NewServiceContext) handles cleanup on error.
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}
	// Use values from the main config struct (passed into NewServiceContext)
	// Use values from the main config struct 'c' passed into this function
	maxIdleConns := 10 // Default
	if c.DBMaxIdleConns > 0 {
		maxIdleConns = c.DBMaxIdleConns
	}
	maxOpenConns := 100 // Default
	if c.DBMaxOpenConns > 0 {
		maxOpenConns = c.DBMaxOpenConns
	}
	connMaxLifetime := time.Hour // Default
	if c.DBConnMaxLifetime > 0 {
		connMaxLifetime = time.Duration(c.DBConnMaxLifetime) * time.Second
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	log.Println("Database connection pool configured.")
	return gormDB, nil
}

// NewServiceContext creates and initializes a new ServiceContext based on the provided configuration.
// It sets up database connections, Redis, NATS, rate limiters, job manager, etc.
// It also runs migrations and seeding after connecting to the database.
func NewServiceContext(c *config.Config) (*ServiceContext, error) {
	// --- Database Initialization & Migration ---
	// Pass the main config 'c' to initGormDB
	gormDB, err := initGormDB(c)
	if err != nil {
		return nil, err // Error already wrapped in initGormDB
	}
	// Run migrations and seeding *after* successful connection
	if err := db.MigrateAndSeed(gormDB); err != nil {
		// Attempt to close DB connection if migration fails
		if sqlDB, dbErr := gormDB.DB(); dbErr == nil {
			sqlDB.Close()
		}
		return nil, fmt.Errorf("database migration/seeding failed: %w", err)
	}

	// --- Redis Initialization ---
	var redisClient *redis.Client
	if c.Redis.URL != "" {
		redisClient, err = cache.InitRedis(c.Redis.URL) // Assuming cache.InitRedis takes URL
		if err != nil {
			log.Printf("WARN: Failed to initialize Redis at %s: %v. Proceeding without cache.", c.Redis.URL, err)
			// Proceed without Redis, redisClient remains nil
		}
	} else {
		log.Println("INFO: Redis URL not provided. Proceeding without cache.")
	}

	// --- Rate Limiter Initialization ---
	// TODO: Make instance count configurable or provide a way to update it post-init. Defaulting to 1.
	instanceCount := 1
	log.Printf("INFO: Initializing rate limiter with default instance count: %d", instanceCount)
	// Use a default base config if specific limiters aren't defined
	baseRateLimiterConfig := ratelimiter.RateLimiterConfig{
		// Define sensible defaults or require base limits in config.Config
		TotalRPM:       1000, // Example default
		MaxConcurrency: 100,  // Example default
		InstanceCount:  instanceCount,
	}
	rateLimiter := ratelimiter.NewRateLimiter(baseRateLimiterConfig)
	// Apply specific limits from config
	for name, limitCfg := range c.RateLimiters {
		log.Printf("INFO: Applying rate limits for '%s'", name)
		rateLimiter.UpdateLimits(&limitCfg, instanceCount) // Pass instanceCount
	}
	// Removed: rateLimiter.StartInstanceCountUpdater - K8s logic is app-specific.

	// --- Job Manager Initialization ---
	jobManager := jobs.NewJobManager() // Assuming jobs.NewJobManager requires no args initially

	// --- PubSub Broker Initialization ---
	var pubSubBroker pubsub.Broker
	if c.Nats.URL != "" {
		// Assuming NewNATSBroker requires both NATS URL and Redis URL (for locking/coordination)
		redisURL := ""
		if redisClient != nil {
			redisURL = c.Redis.URL // Pass Redis URL only if Redis is successfully initialized
		}
		broker, err := pubsub.NewNATSBroker(c.Nats.URL, redisURL)
		if err != nil {
			log.Printf("WARN: Failed to create NATS broker (NATS: %s, Redis: %s): %v. Using no-op broker.", c.Nats.URL, redisURL, err)
			pubSubBroker = &events.NoOpBroker{}
		} else {
			log.Printf("INFO: NATS broker initialized (NATS: %s, Redis: %s)", c.Nats.URL, redisURL)
			pubSubBroker = broker
		}
	} else {
		log.Println("INFO: NATS URL not provided. Using no-op broker.")
		pubSubBroker = &events.NoOpBroker{}
	}

	// // --- Communication Service Initialization ---
	// var commClient communication.Service
	// // Check if AWS config is provided and valid enough to attempt init
	// if c.AWS.Region != "" { // Basic check, InitService might do more validation
	// 	commClient, err = communication.InitService(context.Background(), c.AWS) // Pass AWS config struct
	// 	if err != nil {
	// 		log.Printf("WARN: Failed to initialize communication service (AWS Region: %s): %v", c.AWS.Region, err)
	// 		// Proceed without communication client, commClient remains nil
	// 	} else {
	// 		log.Printf("INFO: Communication service initialized (AWS Region: %s)", c.AWS.Region)
	// 	}
	// } else {
	// 	log.Println("INFO: AWS Region not provided in config. Skipping communication service initialization.")
	// }

	// --- Email Sender Initialization Removed ---
	// The consuming application is responsible for initializing its own email sender.

	// --- Session Manager Initialization ---
	sessionManager := session.NewSession(c) // Assuming session.NewSession takes *config.Config
	// --- AI/MCP Client Initialization (Placeholders) ---
	// TODO: Initialize actual clients based on c.GPT, c.Anthropic, c.MCP etc.
	// var openAIClient, claudeClient, geminiClient interface{} // Removed placeholder variables
	// var mcpServer, mcpClient, goMCPServer, goMCPClient interface{} // Removed placeholder variables
	log.Println("INFO: Skipping core AI/MCP client initialization.") // Updated log message

	// --- Construct ServiceContext ---
	svcCtx := &ServiceContext{
		Config:      c,
		DB:          gormDB, // Assign the initialized DB
		RedisClient: redisClient,
		// CommunicationClient: commClient,
		Session:      sessionManager,
		RateLimiter:  rateLimiter,
		JobManager:   jobManager,
		PubSubBroker: pubSubBroker,
		EventHub:     sse.NewEventHub(), // Assuming sse.NewEventHub needs no args
		// EmailSender:         emailSender, // Removed
		ModuleServices: make(map[string]interface{}), // Initialize empty map
		Settings:       make(models.SettingsMap),     // Initialize with correct type

		// Removed AI/LLM/MCP client assignments
	}

	return svcCtx, nil
}

// ReloadAllSettings reloads all dynamic settings and re-initializes dependent services.
func (svc *ServiceContext) ReloadAllSettings() error {
	// Reload all settings as a map
	settingsMap, err := models.LoadAllSettings(svc.DB) // Uncommented
	if err != nil {                                    // Uncommented
		log.Printf("ERROR: Failed to reload settings from DB: %v", err) // Added logging
		return err                                                      // Uncommented
	} // Uncommented
	svc.Settings = settingsMap                            // Uncommented
	log.Println("INFO: Settings reloaded from database.") // Added logging

	// Re-initialization of specific services (like email, AI clients)
	// is the responsibility of the consuming application or specific modules.
	return nil
}
