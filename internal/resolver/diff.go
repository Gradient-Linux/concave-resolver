package resolver

import (
	"sort"
	"strconv"
	"strings"
)

// CUDAAdjacentPackages lists packages whose version changes are always risky.
var CUDAAdjacentPackages = []string{
	"torch",
	"torchvision",
	"torchaudio",
	"tensorflow",
	"tensorflow-gpu",
	"jax",
	"jaxlib",
	"numpy",
	"cupy",
	"cuda-python",
	"nvidia-cuda-runtime-cu12",
	"nvidia-cudnn-cu12",
	"triton",
}

var cudaAdjacent = func() map[string]struct{} {
	items := make(map[string]struct{}, len(CUDAAdjacentPackages))
	for _, pkg := range CUDAAdjacentPackages {
		items[normalizeName(pkg)] = struct{}{}
	}
	return items
}()

// ParsePipFreeze parses pip freeze output into a package map.
func ParsePipFreeze(output string) map[string]string {
	packages := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "", strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "-e "):
			name := editableName(line)
			if name != "" {
				packages[name] = "editable"
			}
		case strings.Contains(line, " @ "):
			parts := strings.SplitN(line, " @ ", 2)
			if len(parts) == 2 {
				packages[normalizeName(parts[0])] = strings.TrimSpace(parts[1])
			}
		case strings.Contains(line, "=="):
			parts := strings.SplitN(line, "==", 2)
			if len(parts) == 2 {
				packages[normalizeName(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return packages
}

// ClassifyDiff determines the drift tier for a package change.
func ClassifyDiff(packageName, baseline, current string) DriftTier {
	name := normalizeName(packageName)
	if _, ok := cudaAdjacent[name]; ok {
		return DriftLeave
	}
	if baseline == current {
		return DriftSafe
	}

	bParts, bOK := versionParts(baseline)
	cParts, cOK := versionParts(current)
	if !bOK || !cOK {
		return DriftFlag
	}

	switch compareVersionParts(bParts, cParts) {
	case -1:
		return DriftSafe
	case 0:
		return DriftSafe
	case 1:
		switch versionChangeClass(bParts, cParts) {
		case "major":
			return DriftLeave
		case "minor":
			return DriftFlag
		default:
			return DriftSafe
		}
	default:
		return DriftFlag
	}
}

// DiffSnapshots computes package diffs between two snapshots.
func DiffSnapshots(baseline, current Layer3Snapshot) []PackageDiff {
	diffs := make([]PackageDiff, 0)
	for name, baseVersion := range baseline.Packages {
		currentVersion, ok := current.Packages[name]
		if !ok {
			diffs = append(diffs, PackageDiff{
				Name:     name,
				Baseline: baseVersion,
				Tier:     DriftFlag,
				Reason:   "package removed",
			})
			continue
		}
		if baseVersion == currentVersion {
			continue
		}
		tier := ClassifyDiff(name, baseVersion, currentVersion)
		diffs = append(diffs, PackageDiff{
			Name:     name,
			Baseline: baseVersion,
			Current:  currentVersion,
			Tier:     tier,
			Reason:   diffReason(name, baseVersion, currentVersion, tier),
		})
	}
	for name, currentVersion := range current.Packages {
		if _, ok := baseline.Packages[name]; ok {
			continue
		}
		diffs = append(diffs, PackageDiff{
			Name:    name,
			Current: currentVersion,
			Tier:    DriftFlag,
			Reason:  "package added",
		})
	}
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Name < diffs[j].Name
	})
	return diffs
}

func diffReason(name, baseline, current string, tier DriftTier) string {
	switch tier {
	case DriftSafe:
		return "patch version change"
	case DriftFlag:
		if _, ok := cudaAdjacent[normalizeName(name)]; ok {
			return "CUDA-adjacent package"
		}
		return "minor version change"
	case DriftLeave:
		if _, ok := cudaAdjacent[normalizeName(name)]; ok {
			return "CUDA-adjacent package"
		}
		return "major version change"
	default:
		return "unclassified version change"
	}
}

func editableName(line string) string {
	if idx := strings.Index(line, "#egg="); idx >= 0 {
		return normalizeName(line[idx+5:])
	}
	return ""
}

func normalizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func versionParts(version string) ([]int, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if version == "" {
		return nil, false
	}
	fields := strings.FieldsFunc(version, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if len(fields) == 0 {
		return nil, false
	}
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		value, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

func compareVersionParts(baseline, current []int) int {
	max := len(baseline)
	if len(current) > max {
		max = len(current)
	}
	for i := 0; i < max; i++ {
		b := 0
		c := 0
		if i < len(baseline) {
			b = baseline[i]
		}
		if i < len(current) {
			c = current[i]
		}
		switch {
		case c > b:
			return 1
		case c < b:
			return -1
		}
	}
	return 0
}

func versionChangeClass(baseline, current []int) string {
	max := len(baseline)
	if len(current) > max {
		max = len(current)
	}
	for i := 0; i < max; i++ {
		b := 0
		c := 0
		if i < len(baseline) {
			b = baseline[i]
		}
		if i < len(current) {
			c = current[i]
		}
		if c == b {
			continue
		}
		if i == 0 {
			return "major"
		}
		if i == 1 {
			return "minor"
		}
		return "patch"
	}
	return "patch"
}
