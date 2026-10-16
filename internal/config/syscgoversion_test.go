package config

import (
	"strings"
	"testing"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
	"github.com/Nomadcxx/sysc-walls/internal/version"
)

// TestEnforcedMinimumMatchesAdvertised is the other half of #46: the constant
// the daemon actually compares against has to be the one the banner prints.
func TestEnforcedMinimumMatchesAdvertised(t *testing.T) {
	if MinimumSyscGoVersion != version.MinSyscGoVersion {
		t.Errorf("MinimumSyscGoVersion = %q, want %q", MinimumSyscGoVersion, version.MinSyscGoVersion)
	}
	if !strings.HasSuffix(version.SyscGoVersion, MinimumSyscGoVersion) {
		t.Errorf("advertised %q does not carry the enforced minimum %q", version.SyscGoVersion, MinimumSyscGoVersion)
	}
}

func TestMinimumSyscGoVersionMatchesRequiredEffectLibrary(t *testing.T) {
	if got, want := version.MinSyscGoVersion, syscGo.GetLibraryVersion(); got != want {
		t.Errorf("minimum sysc-Go version = %q, built library = %q", got, want)
	}
}
