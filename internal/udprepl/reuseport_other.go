//go:build !darwin

package udprepl

// socketReusePort is unused off darwin: Linux grants the UDP double-bind
// through SO_REUSEADDR alone, and SO_REUSEPORT would additionally let the
// kernel load-balance unicast datagrams between the shared-port sockets.
// See listenDatagram.
const socketReusePort = 0
