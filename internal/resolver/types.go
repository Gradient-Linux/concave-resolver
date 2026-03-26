package resolver

import "time"

// DefaultSocketPath is the Unix socket path used by the resolver daemon.
const DefaultSocketPath = "/run/gradient/resolver.sock"

// SnapshotDir is the directory, relative to the workspace root, where
// environment snapshots are stored.
const SnapshotDir = "config/env-snapshots"

// Layer3Snapshot is a point-in-time record of the Python package state for
// one group. It intentionally excludes hardware-specific layers.
type Layer3Snapshot struct {
	Group     string            `json:"group"`
	Timestamp time.Time         `json:"timestamp"`
	Packages  map[string]string `json:"packages"`
	Backend   string            `json:"backend"`
	MLflowIDs []string          `json:"mlflow_ids"`
}

// DriftTier classifies how severe a package divergence is.
type DriftTier int

const (
	// DriftSafe marks a patch-level change that is safe to auto-apply.
	DriftSafe DriftTier = iota
	// DriftFlag marks a minor version change that should be reviewed.
	DriftFlag
	// DriftLeave marks a major or risky change that should never be auto-applied.
	DriftLeave
)

// String returns the human-readable label for a drift tier.
func (d DriftTier) String() string {
	switch d {
	case DriftSafe:
		return "safe"
	case DriftFlag:
		return "flag"
	case DriftLeave:
		return "leave"
	default:
		return "unknown"
	}
}

// PackageDiff describes a single package's divergence from the baseline.
type PackageDiff struct {
	Name     string    `json:"name"`
	Baseline string    `json:"baseline"`
	Current  string    `json:"current"`
	Tier     DriftTier `json:"tier"`
	Reason   string    `json:"reason"`
}

// DriftReport is the result of a scan for one group.
type DriftReport struct {
	Group     string        `json:"group"`
	User      string        `json:"user"`
	Timestamp time.Time     `json:"timestamp"`
	Diffs     []PackageDiff `json:"diffs"`
	Clean     bool          `json:"clean"`
}

// ResolverStatus is the daemon state exposed over the local socket.
type ResolverStatus struct {
	Running       bool          `json:"running"`
	LastScan      time.Time     `json:"last_scan"`
	GroupReports  []DriftReport `json:"group_reports"`
	SnapshotCount int           `json:"snapshot_count"`
	SocketPath    string        `json:"socket_path"`
}

// StatusRequest asks the socket server for the current resolver state.
type StatusRequest struct {
	Type string `json:"type"`
}

// DriftRequest asks the socket server for drift reports for a group.
type DriftRequest struct {
	Type  string `json:"type"`
	Group string `json:"group"`
}

// ApplyBaselineRequest asks the socket server to promote a snapshot.
type ApplyBaselineRequest struct {
	Type      string `json:"type"`
	Group     string `json:"group"`
	Timestamp string `json:"timestamp"`
}

// BaselineRequest asks the socket server for one promoted baseline.
type BaselineRequest struct {
	Type  string `json:"type"`
	Group string `json:"group"`
}

// Response wraps all server-to-client messages.
type Response struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Target identifies one scan target that belongs to a group.
type Target struct {
	Group     string
	Container string
}

// ServiceConfig controls resolver daemon behavior.
type ServiceConfig struct {
	WorkspaceRoot string
	SocketPath    string
	ScanInterval  time.Duration
	Targets       []Target
}
