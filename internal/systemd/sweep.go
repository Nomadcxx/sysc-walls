package systemd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// screensaverClass is the window class every screensaver is started with
// (config.GetScreensaverCommand).
const screensaverClass = "sysc-walls-screensaver"

// isScreensaverArgv reports whether argv is a screensaver as the daemon
// starts one: a terminal it launches, followed somewhere by --class and the
// screensaver class as their own arguments. A shell, editor or grep whose
// command line merely contains those words does not match, because its
// argv[0] is not the terminal and the words are not separate arguments.
func isScreensaverArgv(argv []string) bool {
	if len(argv) < 3 {
		return false
	}
	switch filepath.Base(argv[0]) {
	case "kitty", "xterm":
	default:
		return false
	}
	for i, arg := range argv[1:] {
		if arg == "--class="+screensaverClass {
			return true
		}
		if arg == "--class" && i+2 < len(argv) && argv[i+2] == screensaverClass {
			return true
		}
	}
	return false
}

// screensaverPIDs lists this user's screensaver processes under procRoot,
// never including self.
func screensaverPIDs(procRoot string, uid, self int) []int {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil
	}
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		dir := filepath.Join(procRoot, e.Name())
		info, err := os.Stat(dir)
		if err != nil {
			continue
		}
		if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != uid {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, "cmdline"))
		if err != nil || len(data) == 0 {
			continue
		}
		if isScreensaverArgv(strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")) {
			pids = append(pids, pid)
		}
	}
	return pids
}
