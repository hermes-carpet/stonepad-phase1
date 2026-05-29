// Package config provides timing constants for the server.
// All server-side timing constants live in this single file.
// See §7.2 and §14 of the Stonepad v1 Implementation Plan.
package config

import "time"

// ServerTiming holds all server-side timing constants.
// These are editable in one place for easy human tuning.
const (
	// HTTPReadTimeout is the maximum duration for reading the entire request.
	HTTPReadTimeout = 10 * time.Second

	// HTTPWriteTimeout is the maximum duration before timing out writes of the response.
	HTTPWriteTimeout = 30 * time.Second

	// HTTPIdleTimeout is the maximum amount of time to wait for the next request.
	HTTPIdleTimeout = 120 * time.Second

	// HTTPRequestTimeout is the per-request timeout.
	HTTPRequestTimeout = 30 * time.Second

	// GracefulShutdownTimeout is how long to wait for in-flight requests during shutdown.
	GracefulShutdownTimeout = 10 * time.Second

	// ForceShutdownTimeout is the hard deadline for shutdown.
	ForceShutdownTimeout = 30 * time.Second

	// DefaultTmpfsSnapshotInterval is the default interval between tmpfs snapshots.
	DefaultTmpfsSnapshotInterval = 300 * time.Second

	// VACUUMThresholdBytes is the minimum DB file size to trigger weekly VACUUM.
	VACUUMThresholdBytes = 10 * 1024 * 1024

	// AuthTokenExpiry is how long session tokens remain valid.
	AuthTokenExpiry = 30 * 24 * time.Hour
)
