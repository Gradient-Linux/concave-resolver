package resolver

import "testing"

func TestParsePipFreezeStandard(t *testing.T) {
	got := ParsePipFreeze("requests==2.31.0\nnumpy==1.26.4\n")
	if got["requests"] != "2.31.0" {
		t.Fatalf("requests version: %q", got["requests"])
	}
	if got["numpy"] != "1.26.4" {
		t.Fatalf("numpy version: %q", got["numpy"])
	}
}

func TestParsePipFreezeEditableAndDirectRef(t *testing.T) {
	got := ParsePipFreeze("-e git+https://example.com/repo.git#egg=localpkg\npackage @ git+https://example.com/pkg.git\n")
	if got["localpkg"] != "editable" {
		t.Fatalf("editable version: %q", got["localpkg"])
	}
	if got["package"] != "git+https://example.com/pkg.git" {
		t.Fatalf("direct ref version: %q", got["package"])
	}
}

func TestClassifyDiffPatchVersionSafe(t *testing.T) {
	if got := ClassifyDiff("requests", "2.31.0", "2.31.1"); got != DriftSafe {
		t.Fatalf("got %v", got)
	}
}

func TestClassifyDiffMinorVersionFlag(t *testing.T) {
	if got := ClassifyDiff("requests", "2.31.0", "2.32.0"); got != DriftFlag {
		t.Fatalf("got %v", got)
	}
}

func TestClassifyDiffMajorVersionLeave(t *testing.T) {
	if got := ClassifyDiff("requests", "2.31.0", "3.0.0"); got != DriftLeave {
		t.Fatalf("got %v", got)
	}
}

func TestClassifyDiffTorchAlwaysLeave(t *testing.T) {
	if got := ClassifyDiff("torch", "2.2.0", "2.2.1"); got != DriftLeave {
		t.Fatalf("got %v", got)
	}
}

func TestClassifyDiffNumpyAlwaysLeave(t *testing.T) {
	if got := ClassifyDiff("numpy", "1.26.0", "2.0.0"); got != DriftLeave {
		t.Fatalf("got %v", got)
	}
}

func TestDiffSnapshotsEmptyDiff(t *testing.T) {
	base := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0"}}
	current := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0"}}
	if got := DiffSnapshots(base, current); len(got) != 0 {
		t.Fatalf("diffs: %#v", got)
	}
}

func TestDiffSnapshotsAddition(t *testing.T) {
	base := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0"}}
	current := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0", "numpy": "1.26.4"}}
	got := DiffSnapshots(base, current)
	if len(got) != 1 || got[0].Name != "numpy" || got[0].Reason != "package added" {
		t.Fatalf("diffs: %#v", got)
	}
}

func TestDiffSnapshotsRemoval(t *testing.T) {
	base := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0", "numpy": "1.26.4"}}
	current := Layer3Snapshot{Packages: map[string]string{"requests": "2.31.0"}}
	got := DiffSnapshots(base, current)
	if len(got) != 1 || got[0].Name != "numpy" || got[0].Reason != "package removed" {
		t.Fatalf("diffs: %#v", got)
	}
}
