package resolver

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// RunCLI executes the resolver command-line interface and returns an exit code.
func RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:], stdout, stderr)
	case "status":
		return statusCommand(args[1:], stdout, stderr)
	case "scan":
		return scanCommand(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspace := fs.String("workspace", DefaultWorkspaceRoot(), "workspace root")
	socketPath := fs.String("socket", DefaultSocketPath, "unix socket path")
	interval := fs.Duration("interval", 5*time.Minute, "scan interval")
	targets := multiTargetFlag{}
	fs.Var(&targets, "target", "scan target in the form group:container")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	service := NewService(ServiceConfig{
		WorkspaceRoot: *workspace,
		SocketPath:    *socketPath,
		ScanInterval:  *interval,
		Targets:       targets.targets,
	}, ContainerScanner{Runner: OSRunner{}})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Run(ctx); err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 1
	}

	return 0
}

func statusCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	socketPath := fs.String("socket", DefaultSocketPath, "unix socket path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	status, err := QueryStatus(*socketPath)
	if err != nil {
		fmt.Fprintf(stderr, "status: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "running=%t last_scan=%s snapshots=%d socket=%s\n",
		status.Running,
		status.LastScan.Format(time.RFC3339),
		status.SnapshotCount,
		status.SocketPath,
	)
	for _, report := range status.GroupReports {
		fmt.Fprintf(stdout, "group=%s clean=%t diffs=%d\n", report.Group, report.Clean, len(report.Diffs))
	}
	return 0
}

func scanCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workspace := fs.String("workspace", DefaultWorkspaceRoot(), "workspace root")
	group := fs.String("group", "default", "group name")
	container := fs.String("container", "", "container name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*container) == "" {
		fmt.Fprintln(stderr, "scan: --container is required")
		return 2
	}

	scanner := ContainerScanner{Runner: OSRunner{}}
	snapshot, err := scanner.ScanContainer(context.Background(), *container, *group)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	if err := SaveSnapshot(*workspace, snapshot); err != nil {
		fmt.Fprintf(stderr, "save: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "%s\n", SnapshotPath(*workspace, snapshot.Group, snapshot.Timestamp))
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "gradient-resolver")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  run     start the resolver daemon")
	fmt.Fprintln(w, "  status  query the local socket")
	fmt.Fprintln(w, "  scan    scan one container and persist a snapshot")
}

type multiTargetFlag struct {
	targets []Target
}

func (m *multiTargetFlag) String() string {
	values := make([]string, 0, len(m.targets))
	for _, target := range m.targets {
		values = append(values, target.Group+":"+target.Container)
	}
	return strings.Join(values, ",")
}

func (m *multiTargetFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("target must be in group:container form")
	}
	m.targets = append(m.targets, Target{
		Group:     strings.TrimSpace(parts[0]),
		Container: strings.TrimSpace(parts[1]),
	})
	return nil
}
