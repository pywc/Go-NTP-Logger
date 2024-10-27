package ntp

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/pywc/Go-NTP-Logger/config"
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
	hours, minutes, seconds := time.Now().Clock()
	newFilename := fmt.Sprintf("%s_%s_%02d%02d%02d.pcap", "output/"+config.OUTPUT_FILE_PREFIX, getCurrentDate(), hours, minutes, seconds)

	return newFilename
}

func (fm *FileManager) RotateFileIfNeeded() error {
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

func (fm *FileManager) LogNTPPacket(packet gopacket.Packet) {
	fm.writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data())

	currentTime := time.Now()
	date := currentTime.Format("2006-01-02")
	hours, minutes, seconds := currentTime.Clock()

	fmt.Printf("[+] %s %02d:%02d:%02d - Logged: %s (NTP version %d)\n", date, hours, minutes, seconds, packet.Addr.IP, version)
}
