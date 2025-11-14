// version.go - Shared version information for sysc-walls
package version

const (
	// Version is the current sysc-walls version
	Version = "1.0.0"

	// Name is the project name
	Name = "sysc-walls"

	// SyscGoVersion is the minimum required sysc-Go version
	SyscGoVersion = "v1.0.1"
)

// GetFullVersion returns version information for sysc-walls
func GetFullVersion() string {
	return Name + " " + Version + " (requires sysc-Go " + SyscGoVersion + "+)"
}
