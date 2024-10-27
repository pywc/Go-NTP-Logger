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
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pywc/Go-NTP-Logger/config"
	"github.com/pywc/Go-NTP-Logger/ip"
	"github.com/pywc/Go-NTP-Logger/ntp"
)

func handleNTPPacket(packet gopacket.Packet, prefixes []*net.IPNet, fm *ntp.FileManager) {
	ipLayerParsed := packet.Layer(layers.LayerTypeIPv4)
	udpLayerParsed := packet.Layer(layers.LayerTypeUDP)
	if ipLayerParsed == nil || udpLayerParsed == nil {
		return
	}
	ipLayer, _ := ipLayerParsed.(*layers.IPv4)
	udpLayer, _ := udpLayerParsed.(*layers.UDP)

	// Ignore packets coming from router addresses or if not valid NTP packets
	isNTP, version := ntp.ParseNTPRecord(udpLayer)
	if ip.ShouldIgnoreIP(ipLayer.SrcIP) || !isNTP {
		return
	}

	// Log if the source IP matches the allowed prefixes
	if ip.IPMatchesPrefixes(ipLayer.SrcIP, prefixes) {
		fm.LogNTPPacket(packet)

		currentTime := time.Now()
		date := currentTime.Format("2006-01-02")
		hours, minutes, seconds := currentTime.Clock()

		fmt.Printf("[+] %s %02d:%02d:%02d - Logged: %s (NTP version %d)\n", date, hours, minutes, seconds, ipLayer.SrcIP.String(), version)
	}
}

// workerPool processes incoming NTP requests using multiple workers.
func workerPool(prefixes []*net.IPNet, fm *ntp.FileManager, packets <-chan gopacket.Packet, wg *sync.WaitGroup) {
	defer wg.Done()
	for packet := range packets {
		handleNTPPacket(packet, prefixes, fm)
	}
}

// startNTPServer initializes the UDP server to handle NTP requests.
func startNTPServer(prefixes []*net.IPNet) {
	device := config.NETWORK_INTERFACE // Replace with your network interface
	snapshotLen := int32(1024)
	promiscuous := false
	timeout := pcap.BlockForever

	handle, err := pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// File manager to switch dates for files
	fm := &ntp.FileManager{}
	fm.RotateFileIfNeeded()

	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Channel for queuing packets
	packets := make(chan gopacket.Packet, 100)

	// Worker pool setup
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerPool(prefixes, fm, packets, &wg)

		// go workerPool(conn, prefixes, fm, packets, &wg)
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

	ipPrefixes, err := ip.LoadPrefixes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error loading IP prefixes: %v\n", err)
		return
	}

	newpath := filepath.Join(".", "output")
	_ = os.MkdirAll(newpath, os.ModePerm)

	startNTPServer(ipPrefixes)
}
