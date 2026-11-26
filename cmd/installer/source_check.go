package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// checkSourceTree refuses to build and install as root from a checkout that
// another account could have changed. The installer compiles whatever tree
// it is started in and copies the result into /usr/local/bin, so a clone in
// a shared, group-writable or synced directory would be root code execution
// for anyone who can write there.
//
// Every directory from dir up to / and every entry in the tree must belong to
// root or to the target user. Nothing may be writable by other accounts,
// except a root-owned sticky ancestor such as /tmp. Group write is allowed
// only for root's group or the user's own primary group, which covers
// distributions that give each user a private group and a 002 umask.
func checkSourceTree(dir string, uid, gid int) error {
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	for d := dir; ; d = filepath.Dir(d) {
		info, err := os.Lstat(d)
		if err != nil {
			return err
		}
		if err := checkSourceEntry(d, info, uid, gid, d != dir); err != nil {
			return err
		}
		if d == filepath.Dir(d) {
			break
		}
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir // not read by the build
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return checkSourceEntry(path, info, uid, gid, false)
	})
}

func checkSourceEntry(path string, info fs.FileInfo, uid, gid int, ancestor bool) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot read the owner of %s", path)
	}
	if st.Uid != 0 && int(st.Uid) != uid {
		return fmt.Errorf("%s belongs to another account (uid %d); build from a checkout you own", path, st.Uid)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		// A symlink's own mode bits mean nothing; the build reads its target.
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		tinfo, err := os.Stat(target)
		if err != nil {
			return err
		}
		return checkSourceEntry(target, tinfo, uid, gid, false)
	}
	perm := info.Mode().Perm()
	if perm&0o002 != 0 && !(ancestor && st.Uid == 0 && info.Mode()&fs.ModeSticky != 0) {
		return fmt.Errorf("%s is writable by every account; build from a checkout only you can change", path)
	}
	if perm&0o020 != 0 && st.Gid != 0 && int(st.Gid) != gid {
		return fmt.Errorf("%s is writable by group %d; build from a checkout only you can change", path, st.Gid)
	}
	return nil
}
