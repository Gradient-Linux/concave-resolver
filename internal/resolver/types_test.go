package resolver

import "testing"

func TestDriftTierString(t *testing.T) {
	cases := map[DriftTier]string{
		DriftSafe:  "safe",
		DriftFlag:  "flag",
		DriftLeave: "leave",
	}
	for tier, want := range cases {
		if got := tier.String(); got != want {
			t.Fatalf("tier %v: got %q want %q", tier, got, want)
		}
	}
}
