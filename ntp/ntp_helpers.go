package ntp

import (
	"encoding/binary"
	"math"
	"net"
	"time"

	"github.com/pywc/Go-NTP-Logger/config"
)

// PacketData holds the data required for each request.
type PacketData struct {
	Addr *net.UDPAddr
	Data []byte
}

// ntpTime calculates the NTP time (seconds since 1900) from the current Unix time (seconds since 1970).
func ntpTime(t time.Time) uint64 {
	seconds := uint64(t.Unix()) + 2208988800 // Convert Unix time to NTP epoch
	fraction := uint64((t.Nanosecond() * int(math.Pow(2, 32))) / 1e9)
	return (seconds << 32) | fraction
}

// parseNTP validates that the incoming UDP packet is an NTP packet based on NTP protocol headers.
func parseNTPPacket(data []byte) (bool, int) {
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

func makeNTPResponse(version int, requestData []byte) []byte {
	response := make([]byte, 48)

	// Set NTP Version and Server Mode
	response[0] = (response[0] & 0xC7) | byte(version<<3)
	response[0] += 4

	// Set Stratum (e.g., 2 for secondary server)
	response[1] = byte(config.NTP_STRATUM)

	// Set Poll Interval (default 4)
	response[2] = byte(config.NTP_POLL_INTERVAL)

	// Precision (arbitrary value, e.g., -10)
	response[3] = byte(config.NTP_PRECISION)

	// Root delay and dispersion (arbitrary values for example)
	binary.BigEndian.PutUint32(response[4:], 0x00000000)
	binary.BigEndian.PutUint32(response[8:], 0x00000003)

	// Reference ID (UCSD GPS IP)
	copy(response[12:16], config.NTP_REF_ID)

	// Set timestamps
	currentTime := time.Now()
	originTime := binary.BigEndian.Uint64(requestData[40:48])
	binary.BigEndian.PutUint64(response[16:], ntpTime(currentTime.Add(-time.Hour))) // Reference Timestamp
	binary.BigEndian.PutUint64(response[24:], originTime)                           // Origin Timestamp
	binary.BigEndian.PutUint64(response[32:], ntpTime(currentTime))                 // Receive Timestamp
	binary.BigEndian.PutUint64(response[40:], ntpTime(currentTime))                 // Transmit Timestamp

	return response
}
