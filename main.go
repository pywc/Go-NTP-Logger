package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pywc/Go-NTP-Logger/config"
	"github.com/pywc/Go-NTP-Logger/ntp"
	"github.com/pywc/Go-NTP-Logger/ip"
)

func handleNTPPacket(packet gopacket.Packet, prefixes []*net.IPNet, fm *ntp.FileManager) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	udpLayer := packet.Layer(layers.LayerTypeUDP)

	if ipLayer == nil || udpLayer == nil {
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)
	udp, _ := udpLayer.(*layers.UDP)

	// Ignore packets coming from router addresses or if not valid NTP packets
	isNTP, version := ntp.ParseNTPPacket(udp)
	if ip.ShouldIgnoreIP(ip.SrcIP.String()) || !isNTP {
		return
	}

	// Log if the source IP matches the allowed prefixes
	if ip.IPMatchesPrefixes(ip.SrcIP, prefixes) {
		fm.LogNTPPacket(packet, version)
	}

	// Send NTP response
	sendNTPResponse(version, packet, ip.SrcIP, ip.por)
}

func sendNTPResponse(version int, packet gopacket.Packet) {
	response := ntp.MakeNTPResponse(version, packet.Data())

	_, err := conn.WriteToUDP(response, addr)
	if err != nil {
		fmt.Printf("[-] Error sending NTP response: %v\n", err)
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

	ipPrefixes, err := prefix.LoadPrefixes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error loading IP prefixes: %v\n", err)
		return
	}

	newpath := filepath.Join(".", "output")
	_ = os.MkdirAll(newpath, os.ModePerm)

	startNTPServer(ipPrefixes)
}
