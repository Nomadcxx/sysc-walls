// version.go - Shared version information for sysc-walls
package version

import (
	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

const (
	// Version is the current sysc-walls version
	Version = "1.0.0"

	// Name is the project name
	Name = "sysc-walls"
)

// GetSyscGoVersion returns the sysc-Go library version
func GetSyscGoVersion() string {
	return syscGo.GetLibraryVersion()
}

// GetFullVersion returns version information for both sysc-walls and sysc-Go
func GetFullVersion() string {
	return Name + " " + Version + " (sysc-Go " + GetSyscGoVersion() + ")"
}
