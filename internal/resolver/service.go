package resolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	if filepath.Base(home) == "gradient" {
		return home
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
	reportsByGroup := make(map[string]DriftReport, len(s.cfg.Targets))
	var scanErr error

	for _, target := range s.cfg.Targets {
		current, err := s.scanner.ScanContainer(ctx, target.Container, target.Group)
		if err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("%s/%s: %w", target.Group, target.Container, err))
			continue
		}

		if err := SaveSnapshot(s.cfg.WorkspaceRoot, current); err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("save snapshot for %s: %w", target.Group, err))
		}
	}

	reports, reportErr := BuildStoredReports(s.cfg.WorkspaceRoot)
	if reportErr != nil {
		scanErr = errors.Join(scanErr, reportErr)
	}
	for _, report := range reports {
		reportsByGroup[report.Group] = report
	}

	count, err := SnapshotCount(s.cfg.WorkspaceRoot)
	if err != nil {
		scanErr = errors.Join(scanErr, err)
	}

	s.setStatus(ResolverStatus{
		Running:       true,
		LastScan:      time.Now().UTC(),
		GroupReports:  flattenReports(reportsByGroup),
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

// PromoteBaseline saves the current snapshot for a group as its active baseline.
func (s *Service) PromoteBaseline(group, timestamp string) (Layer3Snapshot, error) {
	return PromoteBaseline(s.cfg.WorkspaceRoot, group, timestamp)
}

// Baseline returns the promoted baseline snapshot for one group.
func (s *Service) Baseline(group string) (Layer3Snapshot, error) {
	return LoadBaseline(s.cfg.WorkspaceRoot, group)
}

func BuildStoredReports(workspaceRoot string) ([]DriftReport, error) {
	groups, err := knownGroups(workspaceRoot)
	if err != nil {
		return nil, err
	}
	reports := make([]DriftReport, 0, len(groups))
	for _, group := range groups {
		current, err := LatestSnapshot(workspaceRoot, group)
		if err != nil {
			continue
		}
		baseline, baselineErr := LoadBaseline(workspaceRoot, group)
		if baselineErr != nil {
			baseline, baselineErr = PreviousSnapshot(workspaceRoot, group)
		}
		report := DriftReport{
			Group:     group,
			Timestamp: current.Timestamp,
			Clean:     true,
		}
		if baselineErr == nil {
			report.Diffs = DiffSnapshots(baseline, current)
			report.Clean = len(report.Diffs) == 0
		}
		reports = append(reports, report)
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Group < reports[j].Group
	})
	return reports, nil
}

func knownGroups(workspaceRoot string) ([]string, error) {
	set := map[string]struct{}{}

	summaries, err := BaselineSummaries(workspaceRoot)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		set[summary.Group] = struct{}{}
	}

	dir := filepath.Join(workspaceRoot, SnapshotDir)
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read snapshot dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".lock" {
			continue
		}
		parts := strings.SplitN(name, ".", 2)
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		set[parts[0]] = struct{}{}
	}

	groups := make([]string, 0, len(set))
	for group := range set {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	return groups, nil
}

func flattenReports(items map[string]DriftReport) []DriftReport {
	if len(items) == 0 {
		return []DriftReport{}
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	reports := make([]DriftReport, 0, len(keys))
	for _, key := range keys {
		reports = append(reports, items[key])
	}
	return reports
}
