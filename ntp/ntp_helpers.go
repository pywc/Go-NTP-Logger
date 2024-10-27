package ntp

import (
	"math"
	"net"
	"time"

	"github.com/google/gopacket/layers"
)

// PacketData holds the data required for each request.
type PacketData struct {
	Addr *net.UDPAddr
	Data []byte
}

// ntpTime calculates the NTP time (seconds since 1900) from the current Unix time (seconds since 1970).
func NTPTime(t time.Time) uint64 {
	seconds := uint64(t.Unix()) + 2208988800 // Convert Unix time to NTP epoch
	fraction := uint64((t.Nanosecond() * int(math.Pow(2, 32))) / 1e9)
	return (seconds << 32) | fraction
}

// parseNTP validates that the incoming UDP packet is an NTP packet based on NTP protocol headers.
func ParseNTPRecord(udp *layers.UDP) (bool, int) {
	// Check that it is traffic coming to UDP port 123
	if udp.DstPort != 123 {
		return false, 0
	}

	data := udp.Payload

	// Check that the packet has at least the length of a basic NTP packet
	if len(data) < 48 {
		return false, 0
	}

	// NTP packets have a specific structure: first byte contains LI, Version, and Mode
	version := (data[0] >> 3) & 0x07 // Extract bits 3, 4, and 5
	mode := data[0] & 0x07           // Extract the last 3 bits

	// Check if the mode is a valid NTP mode
	if mode < 1 || mode > 5 {
		// Valid NTP modes are 1 (symmetric active) to 5 (broadcast)
		return false, 0
	}

	// Check if the version is within known NTP versions (1 through 4)
	if version < 1 || version > 4 {
		return false, 0
	}

	return true, int(version)
}
