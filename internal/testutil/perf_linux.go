//go:build linux

package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// The kernel files the hardware block reads. Every read is best-effort: a
// missing file degrades its field to perfUnknown and never fails a perf run.
const (
	procCPUInfo   = "/proc/cpuinfo"
	procMemInfo   = "/proc/meminfo"
	procLoadAvg   = "/proc/loadavg"
	procMountInfo = "/proc/self/mountinfo"
	sysBlockPath  = "/sys/block/%s/queue/rotational"
)

// The filesystem magic numbers PerfDumpFSType names: the memory-backed types
// the amplification row must refuse and the disk types the repo runs on.
const (
	magicTmpfs   = 0x01021994
	magicRamfs   = 0x858458f6
	magicExt4    = 0xef53
	magicXFS     = 0x58465342
	magicBtrfs   = 0x9123683e
	magicOverlay = 0x794c7630
)

// The hardware block's own scale constants.
const (
	// kibPerGiB converts /proc/meminfo's KiB total to GiB.
	kibPerGiB = 1024 * 1024

	// revShortLen truncates a VCS revision to the short form a hardware
	// block can carry.
	revShortLen = 12

	// minMemInfoFields is "MemTotal:" plus its value on one meminfo line.
	minMemInfoFields = 2

	// loadAvgFields is the 1, 5 and 15 minute load average triple.
	loadAvgFields = 3

	// minMountInfoFields reaches the mount point (field 4): the row's
	// variable-length optional fields start after the mount options.
	minMountInfoFields = 5
)

// HardwareBlock returns the one-line hardware description every perf run logs
// before it measures: the CPU model, logical CPU count, RAM, load average,
// GOOS/GOARCH and Go version, the module/VCS revision, the race state,
// GOMAXPROCS, and the root filesystem's device, type and rotational flag.
// Every read is best-effort — a missing kernel file degrades one field to
// perfUnknown and never fails a test: the block describes the host, it does not
// gate on it.
func HardwareBlock() string {
	rootDisk := perfUnknown
	if mount, ok := mountFor("/"); ok {
		rootDisk = fmt.Sprintf("%s (%s,rotational=%s)", mount.device, mount.fstype, rotational(mount.device))
	}
	return fmt.Sprintf(
		"hardware: CPU=%s; logical=%d; RAM=%s; load=%s; root-disk=%s; go=%s/%s %s; %s; race=%s; GOMAXPROCS=%d",
		cpuModel(),
		runtime.NumCPU(),
		memTotal(),
		loadAverage(),
		rootDisk,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
		moduleInfo(),
		raceLabel(),
		runtime.GOMAXPROCS(0),
	)
}

// PerfDumpDir resolves the perf run's dump directory: a fresh per-test
// subdirectory under SUPKV_PERF_DUMP_DIR when the operator set it (`make
// perf` points it at $(CURDIR)/.tmp/perf, never a memory-backed /tmp), else
// the test's own TempDir. It logs one line with the path, filesystem type,
// device and rotational flag so a -v run carries the media facts beside the
// hardware block.
func PerfDumpDir(tb testing.TB) string {
	tb.Helper()
	return perfDumpDir(tb, PerfDumpFSType, PerfDumpDevice)
}

// PerfDumpFSType names the filesystem backing path by statfs magic — tmpfs,
// ramfs, ext4, xfs, btrfs or overlayfs, the raw magic for anything else, and
// perfUnknown when statfs fails. The name is what makes a memory-backed dump
// visible to the amplification row. It is never empty.
func PerfDumpFSType(path string) string {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return perfUnknown
	}
	switch stat.Type {
	case magicTmpfs:
		return "tmpfs"
	case magicRamfs:
		return "ramfs"
	case magicExt4:
		return "ext4"
	case magicXFS:
		return "xfs"
	case magicBtrfs:
		return "btrfs"
	case magicOverlay:
		return "overlayfs"
	default:
		return fmt.Sprintf("magic-0x%x", stat.Type)
	}
}

// PerfDumpDevice renders path's backing device and rotational flag as
// <device>,rotational=<0|1|unknown>, or perfUnknown when the path or its mount
// cannot be resolved (never a panic or an empty string).
func PerfDumpDevice(path string) string {
	_, err := os.Stat(path) //nolint:gosec // G703: path is the perf test's own dump directory, never request input.
	if err != nil {
		return perfUnknown
	}
	mount, ok := mountFor(path)
	if !ok || mount.device == "" {
		return perfUnknown
	}
	return fmt.Sprintf("%s,rotational=%s", mount.device, rotational(mount.device))
}

// cpuModel returns the first "model name" line from /proc/cpuinfo, or
// perfUnknown when the file or the field is absent.
func cpuModel() string {
	data, err := os.ReadFile(procCPUInfo)
	if err != nil {
		return perfUnknown
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		key, value, found := strings.Cut(line, ":")
		if found && strings.TrimSpace(key) == "model name" {
			return strings.TrimSpace(value)
		}
	}
	return perfUnknown
}

// memTotal renders /proc/meminfo's MemTotal in GiB, or perfUnknown.
func memTotal() string {
	data, err := os.ReadFile(procMemInfo)
	if err != nil {
		return perfUnknown
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < minMemInfoFields {
			break
		}
		kib, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			break
		}
		return fmt.Sprintf("%.1f GiB", kib/kibPerGiB)
	}
	return perfUnknown
}

// loadAverage returns /proc/loadavg's first three fields (1, 5 and 15 minute
// averages), or perfUnknown.
func loadAverage() string {
	data, err := os.ReadFile(procLoadAvg)
	if err != nil {
		return perfUnknown
	}
	fields := strings.Fields(string(data))
	if len(fields) < loadAvgFields {
		return perfUnknown
	}
	return strings.Join(fields[:3], " ")
}

// moduleInfo renders the main module path and its VCS revision/time when the
// build carried them, or "module=unknown".
func moduleInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Path == "" {
		return "module=unknown"
	}
	revision, when := perfUnknown, perfUnknown
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			when = setting.Value
		}
	}
	if len(revision) > revShortLen {
		revision = revision[:revShortLen]
	}
	return fmt.Sprintf("module=%s (vcs %s, %s)", info.Main.Path, revision, when)
}

// mountPoint is one /proc/self/mountinfo row's identity: the mount point, its
// filesystem type and its backing device (the mount source).
type mountPoint struct {
	point  string
	fstype string
	device string
}

// mountFor resolves path's longest matching mount point from
// /proc/self/mountinfo. ok is false when mountinfo is unreadable or no mount
// covers the path.
func mountFor(path string) (mountPoint, bool) {
	data, err := os.ReadFile(procMountInfo)
	if err != nil {
		return mountPoint{}, false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return mountPoint{}, false
	}
	var best mountPoint
	found := false
	for line := range strings.SplitSeq(string(data), "\n") {
		mount, ok := parseMountInfoLine(line)
		if !ok || !pathWithin(abs, mount.point) {
			continue
		}
		if !found || len(mount.point) > len(best.point) {
			best, found = mount, true
		}
	}
	return best, found
}

// parseMountInfoLine parses one mountinfo row: the mount point is field 4,
// and the filesystem type and mount source follow the "-" separator.
func parseMountInfoLine(line string) (mountPoint, bool) {
	fields := strings.Fields(line)
	if len(fields) < minMountInfoFields {
		return mountPoint{}, false
	}
	separator := slices.Index(fields, "-")
	if separator < minMountInfoFields || separator+2 >= len(fields) {
		return mountPoint{}, false
	}
	return mountPoint{
		point:  unescapeMount(fields[4]),
		fstype: fields[separator+1],
		device: fields[separator+2],
	}, true
}

// pathWithin reports whether path is mountPoint or below it (the mount point
// "/" covers everything).
func pathWithin(path, mountPoint string) bool {
	if mountPoint == "/" {
		return strings.HasPrefix(path, "/")
	}
	return path == mountPoint || strings.HasPrefix(path, mountPoint+"/")
}

// unescapeMount decodes mountinfo's octal escapes (space, tab, newline and
// backslash) in a mount point.
func unescapeMount(field string) string {
	return strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	).Replace(field)
}

// rotational reads /sys/block/<basename>/queue/rotational: "0" for a
// non-rotational device, "1" for a spinning one, perfUnknown when the device is
// not a top-level block device (a device-mapper path or a partition has no
// such directory).
func rotational(device string) string {
	data, err := os.ReadFile(fmt.Sprintf(sysBlockPath, filepath.Base(device)))
	if err != nil {
		return perfUnknown
	}
	return strings.TrimSpace(string(data))
}
