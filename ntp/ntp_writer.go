package ntp

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/pywc/Go-NTP-Logger/tree/main/config"
)

type FileManager struct {
	currentDate string
	writer      *pcapgo.Writer
	outputFile  *os.File
	mutex       sync.Mutex
}

func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

func getNewFileName() string {
	files, _ := filepath.Glob("output/" + OUTPUT_FILE_PREFIX + "*.pcap")
	newFilename := fmt.Sprintf("%s-%s.pcap", "output/"+OUTPUT_FILE_PREFIX, getCurrentDate())

	// handle duplicate files by adding current time to suffix
	for _, file := range files {
		if newFilename == file {
			hours, minutes, seconds := time.Now().Clock()

			newFilename = fmt.Sprintf("%s-%s_%02d%02d%02d.pcap", "output/"+OUTPUT_FILE_PREFIX, getCurrentDate(), hours, minutes, seconds)
		}
	}

	return newFilename
}

func (fm *FileManager) rotateFileIfNeeded() error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	current := getCurrentDate()
	if fm.currentDate != current {
		if fm.outputFile != nil {
			fm.outputFile.Close()
		}

		newFilename := getNewFileName()

		var err error
		fm.outputFile, err = os.Create(newFilename)
		if err != nil {
			return err
		}

		fm.writer = pcapgo.NewWriter(fm.outputFile)
		if err := fm.writer.WriteFileHeader(65536, layers.LinkTypeIPv4); err != nil {
			return err
		}

		fm.currentDate = current
		fmt.Println("[*] Created new packet file:", newFilename)
	}
	return nil
}

func (fm *FileManager) WritePacket(packet gopacket.CaptureInfo, data []byte) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.writer.WritePacket(packet, data)
}

// We need this because of relative timestamps in packet;
// without this, all first packet in file will have a timestamp of zero
func (fm *FileManager) logDummyPacket() {
	// Create IPv4 and UDP layers
	ipLayer := &layers.IPv4{
		Version: 4,
		IHL:     5,
		SrcIP:   net.IPv4(127, 0, 0, 1),
		DstIP:   net.IPv4(127, 0, 0, 1),
		TTL:     64,
	}

	// Prepare gopacket serialization buffer
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buffer, options,
		ipLayer,
	)
	if err != nil {
		fmt.Printf("[-] Error serializing packet: %v\n", err)
		return
	}

	// Write the packet data to the pcap file
	captureInfo := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(buffer.Bytes()),
		Length:        len(buffer.Bytes()),
	}
	fm.WritePacket(captureInfo, buffer.Bytes())
}

func logNTPPacket(packet PacketData, version int, writer *pcapgo.Writer) {
	// Create IPv4 and UDP layers
	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		SrcIP:    packet.Addr.IP,
		DstIP:    net.ParseIP(SERVER_IP),
		Protocol: layers.IPProtocolUDP,
		TTL:      64,
	}
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(packet.Addr.Port),
		DstPort: layers.UDPPort(SERVER_PORT),
	}
	udpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Prepare gopacket serialization buffer
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buffer, options,
		ipLayer,
		udpLayer,
		gopacket.Payload(packet.Data),
	)
	if err != nil {
		fmt.Printf("[-] Error serializing packet: %v\n", err)
		return
	}

	// Write the packet data to the pcap file
	captureInfo := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(buffer.Bytes()),
		Length:        len(buffer.Bytes()),
	}
	writer.WritePacket(captureInfo, buffer.Bytes())

	fmt.Printf("[+] Logged NTP packet from IP: %s (version %d)\n", packet.Addr.IP, version)
}
