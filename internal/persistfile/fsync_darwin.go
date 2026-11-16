//go:build darwin

package persistfile

import (
	"os"
	"syscall"
)

// syncDurable flushes f's buffered data to permanent storage. Apple's
// fcntl(2) defines F_FULLFSYNC as doing "the same thing as fsync(2)" and
// then asking the drive to flush all buffered data to the permanent
// storage device — a strict superset of the fsync that Go's File.Sync
// issues on darwin (the stdlib binds Fsync to plain libc_fsync and never
// sends F_FULLFSYNC, verified in the Go 1.27 toolchain source), so the
// fcntl REPLACES File.Sync here rather than wrapping it: a preceding Sync
// would flush the same data twice. Valid for both file and directory
// descriptors — the dump path syncs the temp file's data and, after the
// rename, the directory entry. This changes no test's meaning either way:
// process-kill durability already holds through the kernel cache on plain
// fsync; only power loss distinguishes the two, and no CI test simulates
// power loss.
func syncDurable(f *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), uintptr(syscall.F_FULLFSYNC), 0)
	if errno != 0 {
		return errno
	}
	return nil
}
