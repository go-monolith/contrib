// Package jwt provides JWT authentication middleware for the Mono framework.
//
// This middleware validates JWT tokens in NATS message headers and implements
// the mono.MiddlewareModule interface to wrap Mono framework handlers.
//
// Features:
//   - JWT signature verification (HMAC, RSA, ECDSA)
//   - Standard claims validation (exp, nbf, iat)
//   - Issuer and audience validation
//   - JWKS endpoint support with caching
//   - Context enhancement with validated claims
//   - Support for all 5 Mono handler types
//
// Example usage with static secret:
//
//	import "github.com/go-monolith/contrib/v1/jwt"
//
//	jwtMw, err := jwt.New(
//	    jwt.WithSecret([]byte("my-secret-key")),
//	    jwt.WithExpectedIssuer("my-app"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	app := mono.New()
//	app.Register(jwtMw)
//	app.Start()
//
// Example usage with JWKS endpoint:
//
//	jwtMw, err := jwt.New(
//	    jwt.WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
//	    jwt.WithJWKSCacheTTL(1 * time.Hour),
//	)
package jwt

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-monolith/mono/pkg/types"
)

// Middleware implements the mono.MiddlewareModule interface for JWT authentication.
type Middleware struct {
	name   string
	config *Config

	// JWT validation components
	validator *TokenValidator

	// JWKS cache (nil if using static keys)
	jwksCache *JWKSCache

	// Lifecycle management
	logger         *slog.Logger
	refreshCtx     context.Context
	refreshCancel  context.CancelFunc
}

// New creates a new JWT middleware instance with the given options.
//
// At least one key source must be configured:
//   - WithSecret() for HMAC validation
//   - WithPublicKey() for RSA/ECDSA validation
//   - WithJWKSEndpoint() for dynamic key fetching
//
// Example:
//
//	mw, err := jwt.New(
//	    jwt.WithSecret([]byte("my-secret")),
//	    jwt.WithExpectedIssuer("my-app"),
//	    jwt.WithRequiredClaims("sub", "email"),
//	)
func New(opts ...Option) (*Middleware, error) {
	// Create config
	config := &Config{}

	// Apply all functional options
	for _, opt := range opts {
		opt(config)
	}

	// Apply defaults for any unset values
	applyDefaults(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Initialize key provider based on configuration
	var keyProvider KeyProvider
	var jwksCache *JWKSCache

	if config.JWKSEndpoint != "" {
		// JWKS mode: create JWKS cache and provider
		jwksCache = NewJWKSCache(config.JWKSCacheTTL)
		keyProvider = NewJWKSKeyProvider(
			config.JWKSEndpoint,
			jwksCache,
			config.JWKSRequestTimeout,
		)
	} else if config.SecretProvider != nil {
		// Secret provider mode: create dynamic secret provider for multi-tenant
		keyProvider = NewSecretProviderKeyProvider(config.SecretProvider)
	} else if len(config.Secret) > 0 {
		// HMAC mode: create static provider with secret
		keyProvider = NewStaticKeyProvider(config.Secret)
	} else if config.PublicKey != nil {
		// RSA/ECDSA mode: create static provider with public key
		keyProvider = NewStaticKeyProvider(config.PublicKey)
	}

	// Create logger instance
	logger := slog.Default()

	// Create validator with the key provider and logger
	validator := NewTokenValidator(keyProvider, config, logger)

	// Create middleware instance
	middleware := &Middleware{
		name:      "jwt",
		config:    config,
		validator: validator,
		jwksCache: jwksCache, // nil if using static keys
		logger:    logger,
		// refreshCtx and refreshCancel will be created in Start()
	}

	return middleware, nil
}

// Name returns the middleware name for Mono framework registration.
//
// Example:
//
//	mw, _ := jwt.New(jwt.WithSecret([]byte("secret")))
//	fmt.Println(mw.Name()) // Output: "jwt"
func (m *Middleware) Name() string {
	return "jwt"
}

// Logger returns the logger instance (set by Mono framework).
//
// Example:
//
//	mw, _ := jwt.New(jwt.WithSecret([]byte("secret")))
//	logger := mw.Logger()
//	logger.Info("JWT middleware initialized")
func (m *Middleware) Logger() *slog.Logger {
	return m.logger
}

// Validator returns the token validator for direct use by modules.
//
// This allows modules to access the validator directly when they need to
// validate JWT tokens outside of the middleware wrapper (e.g., in background
// jobs or custom handlers that need fine-grained control over authentication).
//
// Example:
//
//	validator := jwtMiddleware.Validator()
//	token, err := validator.Extract(msg)
//	if err != nil {
//	    return err
//	}
//	claims, err := validator.Validate(ctx, token)
func (m *Middleware) Validator() *TokenValidator {
	return m.validator
}

// Start initializes the middleware and performs startup tasks.
//
// For JWKS mode, this fetches the initial key set and starts background refresh.
//
// Example:
//
//	mw, _ := jwt.New(
//	    jwt.WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
//	)
//	if err := mw.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
func (m *Middleware) Start(ctx context.Context) error {
	// Create refresh context for background goroutine lifecycle management
	m.refreshCtx, m.refreshCancel = context.WithCancel(context.Background())

	// Check if using JWKS mode
	if m.jwksCache != nil {
		// JWKS mode: perform initial fetch
		m.logger.Info("starting JWT middleware in JWKS mode",
			"endpoint", m.config.JWKSEndpoint,
		)

		// Get the JWKS provider from the validator
		jwksProvider, ok := m.validator.keyProvider.(*JWKSKeyProvider)
		if !ok {
			return ErrJWKSFetchFailed
		}

		// Perform initial JWKS fetch (fail fast if it fails)
		if err := jwksProvider.RefreshCache(ctx); err != nil {
			m.logger.Error("initial JWKS fetch failed", "error", err)
			return err
		}

		m.logger.Info("initial JWKS fetch succeeded")

		// Start background refresh if refresh interval is configured
		if m.config.JWKSRefreshInterval > 0 {
			go m.startBackgroundRefresh()
		}
	} else {
		// Static key mode (HMAC or RSA/ECDSA)
		mode := "HMAC"
		if m.config.PublicKey != nil {
			mode = "RSA/ECDSA"
		}

		m.logger.Info("starting JWT middleware in static key mode",
			"mode", mode,
		)
	}

	return nil
}

// Stop gracefully shuts down the middleware.
//
// This cancels background refresh goroutines and waits for them to exit.
//
// Example:
//
//	mw, _ := jwt.New(jwt.WithSecret([]byte("secret")))
//	if err := mw.Stop(ctx); err != nil {
//	    log.Printf("Error stopping middleware: %v", err)
//	}
func (m *Middleware) Stop(ctx context.Context) error {
	// Cancel the refresh context to signal background goroutines to stop
	if m.refreshCancel != nil {
		m.refreshCancel()
	}

	m.logger.Info("JWT middleware stopped")

	return nil
}

// startBackgroundRefresh starts a background goroutine to periodically refresh JWKS cache.
//
// This method should be called from Start() if using JWKS mode and JWKSRefreshInterval > 0.
// The goroutine will refresh the JWKS cache at the configured interval until the context
// is cancelled via Stop().
//
// The goroutine uses the middleware's refreshCtx and refreshCancel for lifecycle management.
// It logs success/failure of each refresh attempt.
//
// Requirements: FR8 (Background JWKS refresh)
func (m *Middleware) startBackgroundRefresh() {
	// Create a ticker for periodic refresh
	ticker := time.NewTicker(m.config.JWKSRefreshInterval)
	defer ticker.Stop()

	m.logger.Info("background JWKS refresh started",
		"interval", m.config.JWKSRefreshInterval,
	)

	for {
		select {
		case <-m.refreshCtx.Done():
			// Context cancelled - stop the goroutine
			m.logger.Info("background JWKS refresh stopped")
			return

		case <-ticker.C:
			// Time to refresh
			m.logger.Debug("performing background JWKS refresh")

			// Get the JWKS provider from the validator
			if jwksProvider, ok := m.validator.keyProvider.(*JWKSKeyProvider); ok {
				if err := jwksProvider.RefreshCache(m.refreshCtx); err != nil {
					m.logger.Error("background JWKS refresh failed",
						"error", err,
					)
				} else {
					m.logger.Debug("background JWKS refresh succeeded")
				}
			} else {
				// Should not happen - this method should only be called in JWKS mode
				m.logger.Warn("background refresh called but not using JWKS provider")
				return
			}
		}
	}
}

// OnServiceRegistration wraps service handlers with JWT validation logic.
//
// This hook intercepts service registration and wraps handlers based on service type.
// Handlers are wrapped with JWT validation unless the service matches a SkipPaths pattern.
//
// Supported service types:
//   - RequestReply: Wraps RequestHandler
//   - QueueGroup: Wraps each handler in QueueHandlers array
//   - StreamConsumer: Wraps StreamHandler
//
// Event consumer handlers (EventConsumer, EventStreamConsumer) are wrapped via
// separate hooks: OnEventConsumerRegistration and OnEventStreamConsumerRegistration.
//
// Requirements: FR9 (Mono integration), FR10 (Skip paths)
func (m *Middleware) OnServiceRegistration(ctx context.Context, reg types.ServiceRegistration) types.ServiceRegistration {
	// Check if this service should skip JWT validation
	if m.shouldSkip(reg.ModuleName, reg.Name) {
		m.logger.Debug("skipping JWT validation for service",
			"module", reg.ModuleName,
			"service", reg.Name,
		)
		return reg
	}

	// Wrap handlers based on service type
	switch reg.Type {
	case types.ServiceTypeRequestReply:
		if reg.RequestHandler != nil {
			reg.RequestHandler = m.wrapRequestReplyHandler(reg.RequestHandler, reg.ModuleName, reg.Name)
		}

	case types.ServiceTypeQueueGroup:
		if len(reg.QueueHandlers) > 0 {
			wrapped := make([]types.QGHP, len(reg.QueueHandlers))
			for i, pair := range reg.QueueHandlers {
				wrapped[i] = types.QGHP{
					QueueGroup: pair.QueueGroup,
					Handler:    m.wrapQueueGroupHandler(pair.Handler, reg.ModuleName, reg.Name),
				}
			}
			reg.QueueHandlers = wrapped
		}

	case types.ServiceTypeStreamConsumer:
		// Batch handlers are not automatically wrapped by the middleware.
		// Developers must validate each message individually within their handler
		// using the exposed Validator() method, as batch processing should not
		// assume all messages share the same authentication context.
		//
		// Example:
		//   validator := jwtMiddleware.Validator()
		//   for _, msg := range msgs {
		//       token, err := validator.Extract(msg)
		//       claims, err := validator.Validate(ctx, token)
		//       // process message with claims
		//   }
		m.logger.Debug("StreamConsumer handlers require manual JWT validation",
			"module", reg.ModuleName,
			"service", reg.Name,
		)
	}

	return reg
}

// shouldSkip checks if a service should skip JWT validation based on SkipPaths configuration.
//
// This method checks three patterns against the SkipPaths configuration:
//  1. "moduleName.serviceName" - exact match of module and service
//  2. "moduleName" - match all services in the module
//  3. "serviceName" - match this service name in any module
//
// If any pattern matches an entry in SkipPaths, returns true.
//
// Parameters:
//   - moduleName: The name of the module (e.g., "auth", "user")
//   - serviceName: The name of the service (e.g., "Login", "GetUser")
//
// Returns true if the service should skip JWT validation.
//
// Example SkipPaths configurations:
//   - ["auth.Login"] - Skip only the Login service in the auth module
//   - ["auth"] - Skip all services in the auth module
//   - ["Health"] - Skip the Health service in all modules
//   - ["auth.Login", "Health"] - Skip Login in auth and Health in all modules
//
// Requirements: FR10 (Skip paths support)
func (m *Middleware) shouldSkip(moduleName, serviceName string) bool {
	// Build the patterns to check
	fullPath := moduleName + "." + serviceName

	// Check each skip path pattern
	for _, skipPath := range m.config.SkipPaths {
		if skipPath == fullPath || skipPath == moduleName || skipPath == serviceName {
			return true
		}
	}

	return false
}

// OnEventConsumerRegistration wraps event consumer handlers with JWT validation logic.
//
// This hook intercepts event consumer registration and wraps the handler unless
// the event consumer matches a SkipPaths pattern.
//
// Requirements: FR9 (Mono integration), FR10 (Skip paths)
func (m *Middleware) OnEventConsumerRegistration(ctx context.Context, entry types.EventConsumerEntry) types.EventConsumerEntry {
	if entry.Handler == nil {
		return entry
	}

	// Get module and event names for skip check
	moduleName := entry.Module.Name()
	eventName := entry.EventDef.Name

	// Check if this event consumer should skip JWT validation
	if m.shouldSkip(moduleName, eventName) {
		m.logger.Debug("skipping JWT validation for event consumer",
			"module", moduleName,
			"event", eventName,
		)
		return entry
	}

	// Wrap the handler
	entry.Handler = m.wrapEventConsumerHandler(entry.Handler, moduleName, eventName)

	return entry
}

// OnEventStreamConsumerRegistration does not wrap event stream consumer handlers.
//
// Batch event handlers are not automatically wrapped by the middleware because
// validating only the first message in a batch is dangerous - it assumes all
// messages share the same authentication context, which may not be true.
//
// Developers must validate each message individually within their handler using
// the exposed Validator() method.
//
// Example:
//
//	validator := jwtMiddleware.Validator()
//	for _, msg := range msgs {
//	    token, err := validator.Extract(msg)
//	    if err != nil {
//	        // handle missing/invalid token
//	        continue
//	    }
//	    claims, err := validator.Validate(ctx, token)
//	    if err != nil {
//	        // handle validation error
//	        continue
//	    }
//	    // process message with authenticated claims
//	}
//
// Requirements: FR9 (Mono integration)
func (m *Middleware) OnEventStreamConsumerRegistration(ctx context.Context, entry types.EventStreamConsumerEntry) types.EventStreamConsumerEntry {
	if entry.Handler != nil {
		moduleName := entry.Module.Name()
		eventName := entry.EventDef.Name

		m.logger.Debug("EventStreamConsumer handlers require manual JWT validation",
			"module", moduleName,
			"event", eventName,
		)
	}

	return entry
}

// Handler wrapper functions (to be implemented in tasks 23-24)

// wrapRequestReplyHandler wraps a RequestReplyHandler with JWT validation logic.
//
// The wrapper follows this flow:
//  1. Extract JWT token from message headers
//  2. If extraction fails and Optional=true, call original handler without validation
//  3. If extraction fails and Optional=false, return error
//  4. Validate token using the validator
//  5. If validation succeeds, add claims to context and call original handler
//  6. If validation fails, log warning and return error
//
// Requirements: FR1, FR2, FR3, FR4, FR5, FR6, FR9
func (m *Middleware) wrapRequestReplyHandler(handler types.RequestReplyHandler, moduleName, serviceName string) types.RequestReplyHandler {
	return func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		// Extract token from message headers
		token, err := m.validator.Extract(msg)
		if err != nil {
			// Token extraction failed
			if m.config.Optional {
				// Optional mode: allow request without token
				m.logger.Debug("JWT token not found, allowing request in optional mode",
					"module", moduleName,
					"service", serviceName,
					"error", err,
				)
				return handler(ctx, msg)
			}

			// Required mode: reject request
			m.logger.Warn("JWT token extraction failed",
				"module", moduleName,
				"service", serviceName,
				"error", err,
			)
			return nil, err
		}

		// Validate token
		claims, err := m.validator.Validate(ctx, token)
		if err != nil {
			// Validation failed
			m.logger.Warn("JWT validation failed",
				"module", moduleName,
				"service", serviceName,
				"error", err,
			)
			return nil, err
		}

		// Validation succeeded
		m.logger.Debug("JWT validation succeeded",
			"module", moduleName,
			"service", serviceName,
			"subject", claims["sub"],
		)

		// Add claims to context
		ctx = WithClaims(ctx, claims)

		// Call original handler with enhanced context
		return handler(ctx, msg)
	}
}

// wrapQueueGroupHandler wraps a QueueGroupHandler with JWT validation logic.
//
// The wrapper follows the same flow as wrapRequestReplyHandler:
//  1. Extract JWT token from message headers
//  2. If extraction fails and Optional=true, call original handler without validation
//  3. If extraction fails and Optional=false, return error
//  4. Validate token using the validator
//  5. If validation succeeds, add claims to context and call original handler
//  6. If validation fails, log warning and return error
//
// Requirements: FR1, FR2, FR3, FR4, FR5, FR6, FR9
func (m *Middleware) wrapQueueGroupHandler(handler types.QueueGroupHandler, moduleName, serviceName string) types.QueueGroupHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		// Extract token from message headers
		token, err := m.validator.Extract(msg)
		if err != nil {
			// Token extraction failed
			if m.config.Optional {
				// Optional mode: allow request without token
				m.logger.Debug("JWT token not found, allowing request in optional mode",
					"module", moduleName,
					"service", serviceName,
					"error", err,
				)
				return handler(ctx, msg)
			}

			// Required mode: reject request
			m.logger.Warn("JWT token extraction failed",
				"module", moduleName,
				"service", serviceName,
				"error", err,
			)
			return err
		}

		// Validate token
		claims, err := m.validator.Validate(ctx, token)
		if err != nil {
			// Validation failed
			m.logger.Warn("JWT validation failed",
				"module", moduleName,
				"service", serviceName,
				"error", err,
			)
			return err
		}

		// Validation succeeded
		m.logger.Debug("JWT validation succeeded",
			"module", moduleName,
			"service", serviceName,
			"subject", claims["sub"],
		)

		// Add claims to context
		ctx = WithClaims(ctx, claims)

		// Call original handler with enhanced context
		return handler(ctx, msg)
	}
}


// wrapEventConsumerHandler wraps an EventConsumerHandler with JWT validation logic.
//
// The wrapper follows the same flow as wrapRequestReplyHandler:
//  1. Extract JWT token from message headers
//  2. If extraction fails and Optional=true, call original handler without validation
//  3. If extraction fails and Optional=false, return error
//  4. Validate token using the validator
//  5. If validation succeeds, add claims to context and call original handler
//  6. If validation fails, log warning and return error
//
// Requirements: FR1, FR2, FR3, FR4, FR5, FR6, FR9
func (m *Middleware) wrapEventConsumerHandler(handler types.EventConsumerHandler, moduleName, eventName string) types.EventConsumerHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		// Extract token from message headers
		token, err := m.validator.Extract(msg)
		if err != nil {
			// Token extraction failed
			if m.config.Optional {
				// Optional mode: allow request without token
				m.logger.Debug("JWT token not found, allowing request in optional mode",
					"module", moduleName,
					"event", eventName,
					"error", err,
				)
				return handler(ctx, msg)
			}

			// Required mode: reject request
			m.logger.Warn("JWT token extraction failed",
				"module", moduleName,
				"event", eventName,
				"error", err,
			)
			return err
		}

		// Validate token
		claims, err := m.validator.Validate(ctx, token)
		if err != nil {
			// Validation failed
			m.logger.Warn("JWT validation failed",
				"module", moduleName,
				"event", eventName,
				"error", err,
			)
			return err
		}

		// Validation succeeded
		m.logger.Debug("JWT validation succeeded",
			"module", moduleName,
			"event", eventName,
			"subject", claims["sub"],
		)

		// Add claims to context
		ctx = WithClaims(ctx, claims)

		// Call original handler with enhanced context
		return handler(ctx, msg)
	}
}

// Pass-through hooks (no modification needed for JWT middleware)

// OnModuleLifecycle passes through module lifecycle events unchanged.
//
// The JWT middleware doesn't need to observe or modify module lifecycle events.
func (m *Middleware) OnModuleLifecycle(ctx context.Context, event types.ModuleLifecycleEvent) types.ModuleLifecycleEvent {
	return event
}

// OnConfigurationChange passes through configuration events unchanged.
//
// The JWT middleware doesn't need to observe or modify configuration changes.
func (m *Middleware) OnConfigurationChange(ctx context.Context, event types.ConfigurationEvent) types.ConfigurationEvent {
	return event
}

// OnOutgoingMessage passes through outgoing messages unchanged.
//
// The JWT middleware doesn't need to inject or modify outgoing messages.
// (Unlike authentication tokens which are typically only validated on incoming requests)
func (m *Middleware) OnOutgoingMessage(octx types.OutgoingMessageContext) types.OutgoingMessageContext {
	return octx
}
