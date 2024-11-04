package ntp

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

type FileManager struct {
	currentDate string
	writer      *pcapgo.Writer
	outputPcap  *os.File
	outputCsv   *os.File
	mutex       sync.Mutex
}

func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

func getNewFileName(identifier string) (string, string) {
	hours, minutes, seconds := time.Now().Clock()
	newPcap := fmt.Sprintf("%s_%s_%02d%02d%02d.pcap", "output/"+identifier, getCurrentDate(), hours, minutes, seconds)
	newCsv := fmt.Sprintf("%s_%s_%02d%02d%02d.csv", "output/"+identifier, getCurrentDate(), hours, minutes, seconds)

	return newPcap, newCsv
}

func (fm *FileManager) RotateFileIfNeeded(identifier string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	current := getCurrentDate()
	if fm.currentDate != current {
		if fm.outputPcap != nil {
			fm.outputPcap.Close()
		}
		if fm.outputCsv != nil {
			fm.outputCsv.Close()
		}

		newPcap, newCsv := getNewFileName(identifier)

		var err error
		fm.outputPcap, err = os.Create(newPcap)
		if err != nil {
			return err
		}

		fm.outputCsv, err = os.OpenFile(newCsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fm.outputCsv.WriteString("timestamp,ip\n")

		fm.writer = pcapgo.NewWriter(fm.outputPcap)
		if err := fm.writer.WriteFileHeader(65536, layers.LinkTypeIPv4); err != nil {
			return err
		}

		fm.currentDate = current
		fmt.Println("[*] Created new packet file:", newPcap)
		fmt.Println("[*] Created new CSV file:", newCsv)

	}
	return nil
}

func (fm *FileManager) LogNTPPacket(packet gopacket.Packet, ipString string) {
	// BEcause for some reason go prepends MAC addresses
	data := packet.Data()[14:]
	captureInfo := gopacket.CaptureInfo{
		Timestamp:     packet.Metadata().Timestamp,
		CaptureLength: len(data),
		Length:        len(data),
	}

	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.writer.WritePacket(captureInfo, data)

	timeString := strconv.FormatInt(packet.Metadata().Timestamp.Unix(), 10)
	toWrite := timeString + "," + ipString + "\n"
	fmt.Println(toWrite)
	fm.outputCsv.WriteString(toWrite)
}
