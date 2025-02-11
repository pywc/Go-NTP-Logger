package ntp

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/pywc/Go-NTP-Logger/config"
	"github.com/pywc/Go-NTP-Logger/ip"
)

type FileManager struct {
	currentDate string
	outputCsv   *os.File
	mutex       sync.Mutex
}

func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

func keepUpper32(value uint64) uint64 {
	return value >> 32
}

func getNewFileName(identifier string) string {
	hours, minutes, seconds := time.Now().Clock()
	newCsv := fmt.Sprintf("%s_%s_%02d%02d%02d.csv", "output/"+identifier, getCurrentDate(), hours, minutes, seconds)

	return newCsv
}

func (fm *FileManager) RotateFileIfNeeded(identifier string, prefixes *[]*net.IPNet) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	current := getCurrentDate()
	if fm.currentDate != current {
		if fm.outputCsv != nil {
			fm.outputCsv.Close()
		}

		newCsv := getNewFileName(identifier)

		fm.outputCsv, _ = os.OpenFile(newCsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return err
		// }
		fm.outputCsv.WriteString("region,timestamp,ip,srcport,leap,version,mode,stratum,poll,precision,rootdelay,rootdispersion,refid,reftime,origintime,rxtime,txtime\n")

		// TODO: Handle error
		*prefixes, _ = ip.LoadPrefixes(config.IP_PREFIX_FILE)

		fm.currentDate = current
		fmt.Println("[*] Created new CSV file:", newCsv)
		fmt.Println("[*] IP database updated:", config.IP_PREFIX_FILE)
		fmt.Println(len(*prefixes))
	}
	return nil
}

func (fm *FileManager) LogNTPPacket(packet gopacket.Packet, ipString string) {
	// Extract the network layers
	var ethLayer layers.Ethernet
	var ipLayer layers.IPv4
	var udpLayer layers.UDP
	var ntpLayer layers.NTP

	// Decode layers using gopacket
	parser := gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&ethLayer,
		&ipLayer,
		&udpLayer,
		&ntpLayer,
	)
	decodedLayers := []gopacket.LayerType{}

	// Decode the packet
	if err := parser.DecodeLayers(packet.Data(), &decodedLayers); err != nil {
		fmt.Println("Error decoding packet:", err)
		return
	}

	// Ensure UDP and NTP layers are present
	foundUDP, foundNTP := false, false
	for _, layerType := range decodedLayers {
		if layerType == layers.LayerTypeUDP {
			foundUDP = true
		}
		if layerType == layers.LayerTypeNTP {
			foundNTP = true
		}
	}

	if !foundUDP || !foundNTP {
		fmt.Println("Packet does not contain both UDP and NTP layers")
		return
	}

	// Lock before writing to prevent race conditions
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	timeString := strconv.FormatInt(packet.Metadata().Timestamp.Unix(), 10)

	// Prepare CSV log
	ntpLog := []string{
		config.IDENTIFIER,
		timeString,
		ipString,
		strconv.Itoa(int(udpLayer.SrcPort)),
		strconv.Itoa(int(ntpLayer.LeapIndicator)),
		strconv.Itoa(int(ntpLayer.Version)),
		strconv.Itoa(int(ntpLayer.Mode)),
		strconv.Itoa(int(ntpLayer.Stratum)),
		strconv.Itoa(int(ntpLayer.Poll)),
		strconv.Itoa(int(ntpLayer.Precision)),
		strconv.FormatUint(uint64(ntpLayer.RootDelay), 10),
		strconv.FormatUint(uint64(ntpLayer.RootDispersion), 10),
		strconv.FormatUint(uint64(ntpLayer.ReferenceID), 10),
		strconv.FormatUint(keepUpper32(uint64(ntpLayer.ReferenceTimestamp)), 10),
		strconv.FormatUint(keepUpper32(uint64(ntpLayer.OriginTimestamp)), 10),
		strconv.FormatUint(keepUpper32(uint64(ntpLayer.ReceiveTimestamp)), 10),
		strconv.FormatUint(keepUpper32(uint64(ntpLayer.TransmitTimestamp)), 10),
	}

	// Write CSV
	writer := csv.NewWriter(fm.outputCsv)
	writer.Write(ntpLog)
	writer.Flush()
}
