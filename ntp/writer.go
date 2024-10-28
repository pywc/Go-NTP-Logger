package ntp

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
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

func getNewFileName(identifier string) string {
	hours, minutes, seconds := time.Now().Clock()
	newFilename := fmt.Sprintf("%s_%s_%02d%02d%02d.pcap", "output/"+identifier, getCurrentDate(), hours, minutes, seconds)

	return newFilename
}

func (fm *FileManager) RotateFileIfNeeded(identifier string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	current := getCurrentDate()
	if fm.currentDate != current {
		if fm.outputFile != nil {
			fm.outputFile.Close()
		}

		newFilename := getNewFileName(identifier)

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
	// BEcause for some reason go prepends MAC addresses
	data := packet.Data()[14:]
	captureInfo := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(data),
		Length:        len(data),
	}

	fm.writer.WritePacket(captureInfo, data)
}
