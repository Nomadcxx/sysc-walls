package version

import "testing"

// TestBannerMatchesEnforcedMinimum guards the drift that produced #46: the
// version banner and the enforced minimum were two independent constants, so
// the daemon could advertise v1.0.1 and then refuse to start on 1.0.2.
func TestBannerMatchesEnforcedMinimum(t *testing.T) {
	if got, want := SyscGoVersion, "v"+MinSyscGoVersion; got != want {
		t.Errorf("SyscGoVersion = %q, want %q — the banner and the enforced minimum must be the same value", got, want)
	}
}

func TestGetFullVersionReportsEnforcedMinimum(t *testing.T) {
	full := GetFullVersion()
	if want := "v" + MinSyscGoVersion + "+"; !contains(full, want) {
		t.Errorf("GetFullVersion() = %q, want it to contain %q", full, want)
	}
	if !contains(full, Version) {
		t.Errorf("GetFullVersion() = %q, want it to contain the project version %q", full, Version)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
