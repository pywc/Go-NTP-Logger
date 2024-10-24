package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/pcapgo"
)

func shouldIgnorePacket(addr *net.UDPAddr) bool {
	ip := addr.IP.String()
	// ignore if ntp broadcast from router
	return strings.HasSuffix(ip, ".1")
}

func handleNTPPacket(conn *net.UDPConn, packet PacketData, prefixes []*net.IPNet, writer *pcapgo.Writer) {
	// Ignore packets coming from router addresses or if not valid NTP packets
	isNTP, version := parseNTPPacket(packet.Data)
	if shouldIgnorePacket(packet.Addr) || !isNTP {
		return
	}

	// Check if the source IP matches the allowed prefixes
	if IPMatchesPrefixes(packet.Addr.IP, prefixes) {
		logNTPPacket(packet, version, writer)
	}

	// Send a true NTP response
	sendNTPResponse(conn, version, packet.Addr, packet.Data)
}

func sendNTPResponse(conn *net.UDPConn, version int, addr *net.UDPAddr, requestData []byte) {
	response := makeNTPResponse(version, requestData)

	_, err := conn.WriteToUDP(response, addr)
	if err != nil {
		fmt.Printf("[-] Error sending NTP response: %v\n", err)
	}
}

// workerPool processes incoming NTP requests using multiple workers.
func workerPool(conn *net.UDPConn, prefixes []*net.IPNet, fileManager *FileManager, packets <-chan PacketData, wg *sync.WaitGroup) {
	defer wg.Done()
	for packet := range packets {
		handleNTPPacket(conn, packet, prefixes, fileManager.writer)
	}
}

// startNTPServer initializes the UDP server to handle NTP requests.
func startNTPServer(prefixes []*net.IPNet) {
	fmt.Printf("%s NTP Logger - %s\n\n", SERVER_NAME, SERVER_VERSION)

	addr := net.UDPAddr{
		Port: SERVER_PORT,
		IP:   net.ParseIP(SERVER_IP),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("[-] Error starting UDP server: %v\n", err)
		return
	}
	defer conn.Close()

	// File manager to switch dates for files
	fm := &FileManager{}
	fm.rotateFileIfNeeded()
	fm.logDummyPacket() // needed for correct timestamps

	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Channel for queuing packets
	packets := make(chan PacketData, 100)

	// Worker pool setup
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerPool(conn, prefixes, fm, packets, &wg)
	}

	// Infinitely handle incoming packets
	fmt.Println("[*] Listening for NTP traffic...")
	buffer := make([]byte, 1024)
	for {
		// timeout for rotating file if date changed
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Read packet
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		fm.rotateFileIfNeeded()

		// handle timeout and error
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// do nothing
			} else {
				fmt.Printf("[-] Error reading packet: %v\n", err)
			}

			continue
		}

		packetData := make([]byte, n)
		copy(packetData, buffer[:n])

		// Send request to the worker pool
		packets <- PacketData{Addr: clientAddr, Data: packetData}

	}

	// TODO: unreachable code, need better cleanup
	// close(packets)
	// wg.Wait()
}

func main() {
	ipPrefixes, err := loadPrefixes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error loading IP prefixes: %v\n", err)
		return
	}

	newpath := filepath.Join(".", "packets")
	_ = os.MkdirAll(newpath, os.ModePerm)

	startNTPServer(ipPrefixes)
}
