package prefix

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/pywc/Go-NTP-Logger/config"
)

// loadPrefixes reads IP prefixes from a fixed file "prefixes.txt" and returns a slice of net.IPNet objects.
func loadPrefixes() ([]*net.IPNet, error) {
	file, err := os.Open(IP_PREFIX_FILE)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ipPrefixes []*net.IPNet
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		_, ipNet, err := net.ParseCIDR(scanner.Text())
		if err != nil {
			fmt.Printf("[-] Invalid CIDR: %s\n", scanner.Text())
			continue
		}
		ipPrefixes = append(ipPrefixes, ipNet)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ipPrefixes, nil
}

// IPMatchesPrefixes checks if an IP address matches any of the given IP prefixes.
func IPMatchesPrefixes(ip net.IP, prefixes []*net.IPNet) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}
