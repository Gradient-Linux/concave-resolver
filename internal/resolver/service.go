package resolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Service runs the resolver daemon loop and tracks the current state.
type Service struct {
	cfg     ServiceConfig
	scanner ContainerScanner

	mu     sync.RWMutex
	status ResolverStatus
}

// NewService creates a service with default scanner and configuration values.
func NewService(cfg ServiceConfig, scanner ContainerScanner) *Service {
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = DefaultWorkspaceRoot()
	}
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath
	}
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = 5 * time.Minute
	}
	if scanner.Runner == nil {
		scanner.Runner = OSRunner{}
	}

	return &Service{
		cfg:     cfg,
		scanner: scanner,
		status:  ResolverStatus{SocketPath: cfg.SocketPath},
	}
}

// DefaultWorkspaceRoot returns the standard Gradient workspace path.
func DefaultWorkspaceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "gradient")
	}
	return filepath.Join(home, "gradient")
}

// Run starts the socket server and the periodic scan loop.
func (s *Service) Run(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("service is nil")
	}

	socketErr := make(chan error, 1)
	go func() {
		socketErr <- ServeSocket(ctx, s.cfg.SocketPath, s)
	}()

	s.setRunning(true)
	defer s.setRunning(false)

	if err := s.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		// Keep running even if a scan cycle fails.
	}

	ticker := time.NewTicker(s.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-socketErr:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		case <-ticker.C:
			_ = s.RunOnce(ctx)
		}
	}
}

// RunOnce performs one scan cycle and refreshes daemon status.
func (s *Service) RunOnce(ctx context.Context) error {
	reports := make([]DriftReport, 0, len(s.cfg.Targets))
	var scanErr error

	for _, target := range s.cfg.Targets {
		current, err := s.scanner.ScanContainer(ctx, target.Container, target.Group)
		if err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("%s/%s: %w", target.Group, target.Container, err))
			continue
		}

		previous, prevErr := PreviousSnapshot(s.cfg.WorkspaceRoot, target.Group)
		if prevErr != nil {
			reports = append(reports, DriftReport{
				Group:     target.Group,
				Timestamp: current.Timestamp,
				Clean:     true,
			})
		} else {
			diffs := DiffSnapshots(previous, current)
			reports = append(reports, DriftReport{
				Group:     target.Group,
				Timestamp: current.Timestamp,
				Diffs:     diffs,
				Clean:     len(diffs) == 0,
			})
		}

		if err := SaveSnapshot(s.cfg.WorkspaceRoot, current); err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("save snapshot for %s: %w", target.Group, err))
		}
	}

	count, err := SnapshotCount(s.cfg.WorkspaceRoot)
	if err != nil {
		scanErr = errors.Join(scanErr, err)
	}

	s.setStatus(ResolverStatus{
		Running:       true,
		LastScan:      time.Now().UTC(),
		GroupReports:  reports,
		SnapshotCount: count,
		SocketPath:    s.cfg.SocketPath,
	})

	return scanErr
}

// Status returns a copy of the current resolver status.
func (s *Service) Status() ResolverStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := s.status
	status.GroupReports = append([]DriftReport(nil), status.GroupReports...)
	return status
}

// DriftReports returns the reports for one group, or all reports when the group is empty.
func (s *Service) DriftReports(group string) []DriftReport {
	status := s.Status()
	if group == "" {
		return status.GroupReports
	}
	reports := make([]DriftReport, 0)
	for _, report := range status.GroupReports {
		if report.Group == group {
			reports = append(reports, report)
		}
	}
	return reports
}

func (s *Service) setRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Running = running
	s.status.SocketPath = s.cfg.SocketPath
}

func (s *Service) setStatus(status ResolverStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}
