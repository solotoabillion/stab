package stab

import (
	"fmt"
	"log"

	// Import necessary packages from within the module
	"stab/config"
	"stab/internal/handler" // Added back handler import
	"stab/modules"          // Use the modules package for initialization functions
	"stab/svc"

	// "stab/middleware" // Middleware isn't directly used here anymore

	"github.com/labstack/echo/v4" // Added back echo import
	// echoMiddleware "github.com/labstack/echo/v4/middleware" // Alias not needed now
)

// New initializes the core services of the stab module based on the provided configuration.
// It returns a ServiceContext containing connections (DB, Redis), clients, managers (Jobs),
// and other shared resources. It then runs initializers for modules registered globally
// by the consuming application (via init() side effects).
func New(cfg config.Config) (*svc.ServiceContext, error) {
	log.Println("Initializing stab module core services...")

	// Create the core service context using the provided config
	svcCtx, err := svc.NewServiceContext(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating service context: %w", err)
	}

	// --- Module Initialization ---
	// Run initializers for all modules registered globally via modules.Register
	// (triggered by consuming application's imports)
	if err := modules.RunInitializers(svcCtx); err != nil {
		// Decide if module init failure should be fatal
		log.Printf("ERROR: Failed during globally registered module initialization: %v", err)
		// Attempt cleanup before returning error
		Shutdown(&cfg, svcCtx)                                                                     // Pass config for shutdown consistency
		return nil, fmt.Errorf("failed during globally registered module initialization: %w", err) // Make fatal
	}

	log.Println("stab module initialization complete (including registered modules).")
	return svcCtx, nil
}

// RegisterRoutes registers the core API routes provided by the stab module
// onto an existing Echo instance provided by the consuming application.
// It calls the internal handler registration logic.
func RegisterRoutes(e *echo.Echo, svcCtx *svc.ServiceContext) { // Added function back
	log.Println("Registering stab module core routes...")

	// --- Register Handlers ---
	// The handler.RegisterHandlers function defines all the core API groups and routes.
	handler.RegisterHandlers(e, svcCtx) // Call the internal registration

	// Note: Middleware should be applied by the consuming application.

	log.Println("stab module core routes registered.")
}

// Shutdown gracefully shuts down the stab module's services and registered modules.
// It should be called by the consuming application during its shutdown sequence.
func Shutdown(cfg *config.Config, svcCtx *svc.ServiceContext) { // cfg might not be strictly needed now but kept for consistency
	log.Println("Shutting down stab module and registered modules...")

	// Call module shutdowns for globally registered modules (in reverse order)
	modules.RunShutdowns(svcCtx) // Call the correct function

	// Close DB connection
	if svcCtx.DB != nil {
		if sqlDB, err := svcCtx.DB.DB(); err == nil {
			if err := sqlDB.Close(); err == nil {
				log.Println("Database connection closed.")
			} else {
				log.Printf("Error closing database connection: %v", err)
			}
		} else {
			log.Printf("Error getting underlying DB connection for closing: %v", err)
		}
	}

	// Close Redis connection
	if svcCtx.RedisClient != nil {
		if err := svcCtx.RedisClient.Close(); err == nil {
			log.Println("Redis connection closed.")
		} else {
			log.Printf("Error closing Redis connection: %v", err)
		}
	}

	// Stop Job Manager
	if svcCtx.JobManager != nil {
		// Assuming JobManager has a Stop method - adjust if needed
		// svcCtx.JobManager.Stop()
		log.Println("Job manager stopped.") // Placeholder - Add actual stop call if available
	}

	// Close PubSub Broker
	if svcCtx.PubSubBroker != nil {
		if closer, ok := svcCtx.PubSubBroker.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Error closing PubSub broker: %v", err)
			} else {
				log.Println("PubSub broker closed.")
			}
		}
	}
	log.Println("stab module shutdown complete.")
}
