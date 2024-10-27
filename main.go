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

	// Send NTP response
	// sendNTPResponse(version, udpLayer.Payload, ipLayer.SrcIP, udpLayer.SrcPort)
}

// func sendNTPResponse(version int, payload []byte, dstIP net.IP, dstPort layers.UDPPort) {
// 	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{
// 		IP:   dstIP,
// 		Port: int(dstPort),
// 	})
// 	if err != nil {
// 		log.Println("Failed to connect to destination:", err)
// 		return
// 	}
// 	defer conn.Close()

// 	response := ntp.MakeNTPResponse(version, payload)

// 	_, err = conn.Write(response)
// 	if err != nil {
// 		fmt.Printf("[-] Error sending NTP response: %v\n", err)
// 	}
// }

// workerPool processes incoming NTP requests using multiple workers.
func workerPool(prefixes []*net.IPNet, fm *ntp.FileManager, packets <-chan gopacket.Packet, wg *sync.WaitGroup) {
	defer wg.Done()
	for packet := range packets {
		handleNTPPacket(packet, prefixes, fm)
	}
}

// startNTPServer initializes the UDP server to handle NTP requests.
func startNTPServer(prefixes []*net.IPNet) {
	device := "eth0" // Replace with your network interface
	snapshotLen := int32(1024)
	promiscuous := false
	timeout := pcap.BlockForever

	handle, err := pcap.OpenLive(device, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// addr := net.UDPAddr{
	// 	Port: config.SERVER_PORT,
	// 	IP:   net.ParseIP(config.SERVER_IP),
	// }

	// conn, err := net.ListenUDP("udp", &addr)
	// if err != nil {
	// 	fmt.Printf("[-] Error starting UDP server: %v\n", err)
	// 	return
	// }
	// defer conn.Close()

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

	// buffer := make([]byte, 4096)
	// for {
	// 	// timeout for rotating file if date changed
	// 	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 	// Read packet
	// 	n, clientAddr, err := conn.ReadFromUDP(buffer)
	// 	fm.RotateFileIfNeeded()

	// 	// handle timeout and error
	// 	if err != nil {
	// 		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
	// 			// do nothing
	// 		} else {
	// 			fmt.Printf("[-] Error reading packet: %v\n", err)
	// 		}

	// 		continue
	// 	}

	// 	packetData := make([]byte, n)
	// 	copy(packetData, buffer[:n])

	// 	// Send request to the worker pool
	// 	packets <- ntp.PacketData{Addr: clientAddr, Data: packetData}

	// }

	// TODO: unreachable code, need better cleanup
	// close(packets)
	// wg.Wait()
}

func main() {
	fmt.Printf("%s NTP Logger - %s\n\n", config.SERVER_NAME, config.SERVER_VERSION)

	ipPrefixes, err := ip.LoadPrefixes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error loading IP prefixes: %v\n", err)
		return
	}

	newpath := filepath.Join(".", "output")
	_ = os.MkdirAll(newpath, os.ModePerm)

	startNTPServer(ipPrefixes)
}
