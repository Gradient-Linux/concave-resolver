package resolver

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Runner executes a command and returns its combined output.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSRunner executes commands on the local operating system.
type OSRunner struct{}

// Run executes a command and returns its combined output.
func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// ContainerScanner scans a container by running pip freeze inside it.
type ContainerScanner struct {
	Runner Runner
}

// ScanContainer scans one container and produces a Layer3Snapshot.
func (s ContainerScanner) ScanContainer(ctx context.Context, containerName, group string) (Layer3Snapshot, error) {
	if s.Runner == nil {
		return Layer3Snapshot{}, fmt.Errorf("scanner runner is nil")
	}

	out, err := s.Runner.Run(ctx, "docker", "exec", containerName, "python", "-m", "pip", "freeze")
	if err != nil {
		return Layer3Snapshot{}, fmt.Errorf("scan container %q: %w", containerName, err)
	}

	packages := ParsePipFreeze(string(bytes.TrimSpace(out)))
	return Layer3Snapshot{
		Group:     group,
		Timestamp: time.Now().UTC(),
		Packages:  packages,
		Backend:   "unknown",
		MLflowIDs: nil,
	}, nil
}
