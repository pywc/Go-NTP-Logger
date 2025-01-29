package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/pywc/Go-NTP-Logger/config"
	"github.com/pywc/Go-NTP-Logger/ip"
	"github.com/pywc/Go-NTP-Logger/ntp"
)

// handleNTPPacket processes incoming NTP requests and logs them if the source IP is in the prefix lixt.
func handleNTPPacket(packet gopacket.Packet, prefixes *[]*net.IPNet, fm *ntp.FileManager) {
	ipLayer, udpLayer := ntp.GetLayers(packet)
	if ipLayer == nil || udpLayer == nil {
		return
	}

	// Ignore packets if broadcast coming from router or if not valid NTP
	isNTP, version := ntp.ParseNTPRecord(udpLayer)
	if ntp.ShouldIgnoreIP(ipLayer.SrcIP) || !isNTP {
		return
	}

	// Log if the source IP matches the allowed prefixes
	if ip.IPMatchesPrefixes(ipLayer.SrcIP, prefixes) {
		fm.RotateFileIfNeeded(config.IDENTIFIER, prefixes)
		fm.LogNTPPacket(packet, ipLayer.SrcIP.String())

		currentTime := time.Now()
		date := currentTime.Format("2006-01-02")
		hours, minutes, seconds := currentTime.Clock()

		fmt.Printf("[+] %s %02d:%02d:%02d - Logged: %s (NTP version %d)\n", date, hours, minutes, seconds, ipLayer.SrcIP.String(), version)
	}
}

// workerPool processes incoming NTP requests using multiple workers.
func workerPool(prefixes *[]*net.IPNet, fm *ntp.FileManager, packets <-chan gopacket.Packet, wg *sync.WaitGroup) {
	defer wg.Done()
	for packet := range packets {
		handleNTPPacket(packet, prefixes, fm)
	}
}

// startNTPLogger initializes a packet capture session to handle NTP requests.
func startNTPLogger(prefixes []*net.IPNet) {
	device := config.NETWORK_INTERFACE
	snapshotLen := int32(1024)
	promiscuous := false
	timeout := pcap.BlockForever

	// Start packet capture session
	handle, err := pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// File manager to switch dates for files
	fm := &ntp.FileManager{}

	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Channel for queuing packets
	packets := make(chan gopacket.Packet, 100)

	// Worker pool setup
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerPool(&prefixes, fm, packets, &wg)
	}

	// Infinitely handle incoming packets
	fmt.Println("[*] Listening for NTP traffic...")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		packets <- packet
	}

	// Close channel and wait for all workers to finish
	close(packets)
	wg.Wait()
}

func main() {
	fmt.Printf("Go NTP Logger - %s\n\n", config.IDENTIFIER)

	ipPrefixes, err := ip.LoadPrefixes(config.IP_PREFIX_FILE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error loading IP prefixes: %v\n", err)
		return
	}

	newpath := filepath.Join(".", "output")
	_ = os.MkdirAll(newpath, os.ModePerm)

	startNTPLogger(ipPrefixes)
}
