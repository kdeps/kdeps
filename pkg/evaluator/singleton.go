package evaluator

import (
	"context"
	"fmt"
	"sync"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
)

// EvaluatorManager is a singleton that manages a single PKL evaluator instance
type EvaluatorManager struct {
	evaluator pkl.Evaluator
	context   context.Context
	logger    *logging.Logger
	mu        sync.RWMutex
}

var (
	instance *EvaluatorManager
	once     sync.Once
)

// EvaluatorConfig holds configuration for the evaluator
type EvaluatorConfig struct {
	ResourceReaders []pkl.ResourceReader
	Logger          *logging.Logger
	Options         func(*pkl.EvaluatorOptions)
}

// InitializeEvaluator initializes the singleton evaluator with the provided configuration
func InitializeEvaluator(ctx context.Context, config *EvaluatorConfig) error {
	var err error
	
	// Check if instance exists but evaluator is nil (closed)
	if instance != nil {
		instance.mu.RLock()
		if instance.evaluator == nil {
			instance.mu.RUnlock()
			// Reset the singleton to allow reinitialization
			Reset()
		} else {
			instance.mu.RUnlock()
			// Evaluator already exists and is valid, return success
			return nil
		}
	}
	
	once.Do(func() {
		instance = &EvaluatorManager{
			context: ctx,
			logger:  config.Logger,
		}

		// Set default options if none provided
		opts := config.Options
		if opts == nil {
			opts = func(options *pkl.EvaluatorOptions) {
				pkl.WithDefaultAllowedResources(options)
				pkl.WithOsEnv(options)
				pkl.WithDefaultAllowedModules(options)
				pkl.WithDefaultCacheDir(options)
				options.Logger = pkl.NoopLogger
				options.AllowedModules = []string{".*"}
				options.AllowedResources = []string{".*"}
				options.OutputFormat = "pcf"
			}
		}

		// Add resource readers if provided
		if config.ResourceReaders != nil {
			originalOpts := opts
			opts = func(options *pkl.EvaluatorOptions) {
				originalOpts(options)
				options.ResourceReaders = config.ResourceReaders
			}
		}

		instance.evaluator, err = pkl.NewEvaluator(ctx, opts)
		if err != nil {
			config.Logger.Error("failed to create PKL evaluator", "error", err)
			return
		}

		config.Logger.Debug("PKL evaluator singleton initialized successfully")
	})

	return err
}

// GetEvaluator returns the singleton evaluator instance
func GetEvaluator() (pkl.Evaluator, error) {
	if instance == nil {
		return nil, fmt.Errorf("evaluator not initialized - call InitializeEvaluator first")
	}

	instance.mu.RLock()
	defer instance.mu.RUnlock()

	if instance.evaluator == nil {
		return nil, fmt.Errorf("evaluator instance is nil")
	}

	return instance.evaluator, nil
}

// GetEvaluatorManager returns the singleton evaluator manager instance
func GetEvaluatorManager() (*EvaluatorManager, error) {
	if instance == nil {
		return nil, fmt.Errorf("evaluator manager not initialized - call InitializeEvaluator first")
	}

	return instance, nil
}

// Close closes the singleton evaluator
func (em *EvaluatorManager) Close() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.evaluator != nil {
		err := em.evaluator.Close()
		em.evaluator = nil
		if err != nil {
			em.logger.Error("failed to close PKL evaluator", "error", err)
			return err
		}
		em.logger.Debug("PKL evaluator closed successfully")
	}

	return nil
}

// Reset resets the singleton instance (mainly for testing)
func Reset() {
	if instance != nil {
		instance.Close()
		instance = nil
	}
	once = sync.Once{}
}

// EvaluateModuleSource evaluates a module source and returns the result
func (em *EvaluatorManager) EvaluateModuleSource(ctx context.Context, source *pkl.ModuleSource) (string, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.evaluator == nil {
		return "", fmt.Errorf("evaluator is nil")
	}

	return em.evaluator.EvaluateOutputText(ctx, source)
}
