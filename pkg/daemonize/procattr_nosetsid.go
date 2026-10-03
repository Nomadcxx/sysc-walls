//go:build nosetsid

package daemonize

import "syscall"

// daemonProcAttr without Setsid.
//
// setsid requires CAP_SETSID, which sandboxes and some CI runners do not grant.
// The end-to-end re-exec test builds its helper with the nosetsid tag so the
// launcher/child split can still be exercised where the capability is missing.
// Nothing else in the package is affected: the child still gets its own
// process group, and production builds use the default in procattr.go.
var daemonProcAttr = &syscall.SysProcAttr{
	Setpgid: true,
}
