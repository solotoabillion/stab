package modules

import (
	// Bring back svc import
	"fmt"
	"sync" // Keep mutex for global slice

	"stab/svc"

	"github.com/zeromicro/go-zero/core/logx"
	// Removed config import
	// Removed gorm, redis imports
)

// Initializer allows modules to hook into the application lifecycle.
// Reverted to use *svc.ServiceContext
type Initializer interface {
	// Initialize performs module-specific setup after core services are ready.
	Initialize(svcCtx *svc.ServiceContext) error // Reverted signature
	// Shutdown performs module-specific cleanup before the application exits.
	Shutdown(svcCtx *svc.ServiceContext) error
	// Name returns the unique module name used for registration and configuration.
	Name() string
}

// --- Global Registration (Reverted) ---

var (
	// Global slice to hold initializers registered via init()
	registeredInitializers []Initializer
	mu                     sync.RWMutex // Protects access to registeredInitializers
)

// Register adds an initializer to the global registry.
// This should be called by the init() function of each module's registration package.
// Consuming applications trigger this by importing the module's init package.
func Register(initializer Initializer) { // Renamed back to Register
	mu.Lock()
	defer mu.Unlock()
	name := initializer.Name()
	// Optional: Check for duplicates if desired
	// for _, existing := range registeredInitializers {
	// 	if existing.Name() == name {
	// 		logx.Warnf("Initializer for module '%s' is already registered.", name)
	// 		return // Or panic, depending on desired behavior
	// 	}
	// }
	logx.Infof("Registering global initializer for module: %s", name)
	registeredInitializers = append(registeredInitializers, initializer)
}

// RunInitializers executes the Initialize method for all globally registered modules.
// Called by soul.New().
func RunInitializers(svcCtx *svc.ServiceContext) error {
	mu.RLock() // Read lock
	initializersToRun := make([]Initializer, len(registeredInitializers))
	copy(initializersToRun, registeredInitializers)
	mu.RUnlock() // Unlock before running potentially long initializers

	logx.Infof("Running initializers for %d globally registered modules...", len(initializersToRun))
	for _, initer := range initializersToRun {
		logx.Infof("Initializing module: %s", initer.Name())
		if err := initer.Initialize(svcCtx); err != nil { // Use reverted signature
			logx.Errorf("Failed to initialize module %s: %v", initer.Name(), err)
			// Decide if failure should be fatal
			return fmt.Errorf("failed to initialize module %s: %w", initer.Name(), err) // Fail fast
		}
		// Module service registration (if any) should happen within the module's Initialize method now
		// using svcCtx.ModuleServices directly (or via a helper if preferred).
	}
	logx.Info("Finished running globally registered module initializers.")
	return nil
}

// RunShutdowns executes the Shutdown method for all globally registered modules.
// Called by soul.Shutdown().
func RunShutdowns(svcCtx *svc.ServiceContext) {
	mu.RLock() // Read lock
	initializersToRun := make([]Initializer, len(registeredInitializers))
	copy(initializersToRun, registeredInitializers)
	mu.RUnlock() // Unlock before running potentially long shutdowns

	logx.Info("Running shutdowns for globally registered modules...")
	// Shutdown in reverse order of registration (approximates reverse init)
	for i := len(initializersToRun) - 1; i >= 0; i-- {
		initer := initializersToRun[i]
		logx.Infof("Shutting down module: %s", initer.Name())
		if err := initer.Shutdown(svcCtx); err != nil {
			logx.Errorf("Error shutting down module %s: %v", initer.Name(), err)
			// Continue shutting down other modules even if one fails
		}
	}
	logx.Info("Finished running globally registered module shutdowns.")
}

// Removed ModuleContext interface
