//go:build !nosetsid

package daemonize

import "syscall"

// daemonProcAttr is how the re-executed child is detached: its own session, so
// it has no controlling terminal and cannot acquire one, plus its own process
// group so the two can be signalled separately.
var daemonProcAttr = &syscall.SysProcAttr{
	Setsid:  true,
	Setpgid: true,
}
