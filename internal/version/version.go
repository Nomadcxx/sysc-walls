// version.go - Shared version information for sysc-walls
package version

// MinSyscGoVersion is the oldest sysc-Go release this build works with.
//
// This is the single source of truth: the daemon enforces it before it starts,
// and the -version banner reports it. Keeping one constant is the point — they
// used to be declared separately and drifted apart, so a user on an
// intermediate release was told they met the requirement and then the daemon
// refused to start.
const MinSyscGoVersion = "1.0.3"

// SyscGoVersion is the minimum required sysc-Go version, in the prefixed form
// used by the sysc-Go release tags.
const SyscGoVersion = "v" + MinSyscGoVersion

// Version is the current sysc-walls version.
//
// It is a var rather than a const so release builds can stamp it with
// -ldflags "-X github.com/Nomadcxx/sysc-walls/internal/version.Version=...".
var Version = "1.0.1"

// Name is the project name
const Name = "sysc-walls"

// GetFullVersion returns version information for sysc-walls
func GetFullVersion() string {
	return Name + " " + Version + " (requires sysc-Go " + SyscGoVersion + "+)"
}
