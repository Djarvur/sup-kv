//go:build darwin

package udprepl

import "syscall"

// socketReusePort is the option macOS needs for the test-only same-port
// topology: its kernel refuses a second live UDP bind with SO_REUSEADDR
// alone, so WithAddressReuse sets both there. See listenDatagram.
const socketReusePort = syscall.SO_REUSEPORT
