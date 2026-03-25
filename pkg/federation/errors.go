package federation

import "errors"

// Common federation errors.
var (
	ErrInvalidURN          = errors.New("invalid URN format")
	ErrURNMismatch         = errors.New("URN content hash does not match spec")
	ErrMissingAuthority    = errors.New("URN missing authority")
	ErrInvalidNamespace    = errors.New("URN invalid namespace")
	ErrInvalidName         = errors.New("URN invalid name")
	ErrInvalidVersion      = errors.New("URN invalid version")
	ErrInvalidHash         = errors.New("URN invalid hash format")
	ErrInvalidHashAlgo     = errors.New("URN unsupported hash algorithm")
	ErrRegistryUnreachable = errors.New("registry unreachable")
	ErrAgentNotFound       = errors.New("agent not found in registry")
	ErrCapabilityInvalid   = errors.New("capability schema invalid")
	ErrSignatureInvalid    = errors.New("invalid signature")
	ErrReceiptInvalid      = errors.New("receipt validation failed")
	ErrMissingPublicKey    = errors.New("missing public key for verification")
	ErrAuthFailed          = errors.New("authentication failed")
	ErrUnauthorized        = errors.New("caller not authorized")
	ErrRateLimited         = errors.New("rate limit exceeded")
	ErrTimeout             = errors.New("invocation timed out")
	ErrSchemaMismatch      = errors.New("input does not match agent schema")
	ErrMissingInput        = errors.New("required input fields missing")
	ErrCacheMiss           = errors.New("cache miss")
	ErrTrustLevel          = errors.New("agent trust level insufficient")
	ErrKeyNotFound         = errors.New("key not found in keyring")
	ErrKeyRotation         = errors.New("key rotation in progress")
)

// IsFederationError checks if err is a known federation error.
func IsFederationError(err error) bool {
	if err == nil {
		return false
	}
	targets := []error{
		ErrInvalidURN,
		ErrURNMismatch,
		ErrMissingAuthority,
		ErrInvalidNamespace,
		ErrInvalidName,
		ErrInvalidVersion,
		ErrInvalidHash,
		ErrInvalidHashAlgo,
		ErrRegistryUnreachable,
		ErrAgentNotFound,
		ErrCapabilityInvalid,
		ErrSignatureInvalid,
		ErrReceiptInvalid,
		ErrMissingPublicKey,
		ErrAuthFailed,
		ErrUnauthorized,
		ErrRateLimited,
		ErrTimeout,
		ErrSchemaMismatch,
		ErrMissingInput,
		ErrCacheMiss,
		ErrTrustLevel,
		ErrKeyNotFound,
		ErrKeyRotation,
	}
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

