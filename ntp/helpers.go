package ntp

import (
	"net"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ShouldIgnore IP checks if the IP address is from the router.
func ShouldIgnoreIP(ip net.IP) bool {
	return strings.HasSuffix(ip.String(), ".1")
}

// GetLayers gets the IP and the UDP layers of the packet.
func GetLayers(packet gopacket.Packet) (*layers.IPv4, *layers.UDP) {
	ipLayerParsed := packet.Layer(layers.LayerTypeIPv4)
	udpLayerParsed := packet.Layer(layers.LayerTypeUDP)
	if ipLayerParsed == nil || udpLayerParsed == nil {
		return nil, nil
	}

	ipLayer, _ := ipLayerParsed.(*layers.IPv4)
	udpLayer, _ := udpLayerParsed.(*layers.UDP)

	return ipLayer, udpLayer
}

// ParseNTPRecord validates that the incoming UDP packet is an NTP packet based on NTP protocol headers.
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
