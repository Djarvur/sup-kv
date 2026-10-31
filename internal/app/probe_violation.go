// internal/app/probe_violation.go — NOT committed to master; used in verification:
// copied into place, gate asserted RED, removed, gate asserted GREEN.
package app

import "net"

// ConnNames violates the sup-kv-app-no-io boundary on purpose.
func ConnNames() []string {
	var out []string
	for _, a := range []net.Addr{nil} {
		out = append(out, a.String())
	}
	return out
}
