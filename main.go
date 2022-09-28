package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/gordonklaus/portaudio"
)

func main() {
	fileName := ""
	if len(os.Args) < 2 {
		now := time.Now()

		fileName = fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

		fmt.Println("Using " + fileName + ".mp3 as file name." + "(To specify file name, use arg1)")

		fileName += ".aiff"
	} else {
		fileName = os.Args[1]
	}

	portaudio.Initialize()
	// show the DefaultInputDevice
	inputDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		fmt.Println("Error in getting default input device")
	}
	fmt.Println("Using input device: " + inputDevice.Name + " (host: " + fmt.Sprint(inputDevice.HostApi.Name) + ")")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	if !strings.HasSuffix(fileName, ".aiff") {
		fileName += ".aiff"
	}
	f, err := os.Create(fileName)
	chk(err)

	// form chunk
	_, err = f.WriteString("FORM")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(0))) //total bytes
	_, err = f.WriteString("AIFF")
	chk(err)

	// common chunk
	_, err = f.WriteString("COMM")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(18)))                  //size
	chk(binary.Write(f, binary.BigEndian, int16(1)))                   //channels
	chk(binary.Write(f, binary.BigEndian, int32(0)))                   //number of samples
	chk(binary.Write(f, binary.BigEndian, int16(32)))                  //bits per sample
	_, err = f.Write([]byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0}) //80-bit sample rate 44100
	chk(err)

	// sound chunk
	_, err = f.WriteString("SSND")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(0))) //size
	chk(binary.Write(f, binary.BigEndian, int32(0))) //offset
	chk(binary.Write(f, binary.BigEndian, int32(0))) //block
	nSamples := 0

	fmt.Println("Recording...  Press Ctrl-C to stop.")
	defer portaudio.Terminate()
	in := make([]int32, 64)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in), in)
	chk(err)
	//defer

	chk(stream.Start())
	for {
		chk(stream.Read())
		chk(binary.Write(f, binary.BigEndian, in))
		nSamples += len(in)

		// check <-sig , if it's not nil, then break
		select {
		case <-sig:
			// cancel the signal
			signal.Stop(sig)
			chk(stream.Stop())
			stream.Close()
			// fill in missing sizes
			totalBytes := 4 + 8 + 18 + 8 + 8 + 4*nSamples
			_, err = f.Seek(4, 0)
			chk(err)
			chk(binary.Write(f, binary.BigEndian, int32(totalBytes)))
			_, err = f.Seek(22, 0)
			chk(err)
			chk(binary.Write(f, binary.BigEndian, int32(nSamples)))
			_, err = f.Seek(42, 0)
			chk(err)
			chk(binary.Write(f, binary.BigEndian, int32(4*nSamples+8)))
			chk(f.Close())

			//check if ffmpeg is installed
			_, err = exec.LookPath("ffmpeg")
			if err == nil {
				//fmt.Println("Converting to mp3...")

				cmd := exec.Command("ffmpeg", "-i", fileName, "-ab", "192k", fileName[:len(fileName)-5]+".mp3")
				// fmt.Printf("Command: %s\n", cmd.String())
				//display the output
				//cmd.Stdout = os.Stdout
				//cmd.Stderr = os.Stderr

				err = cmd.Run()
				if err == nil {
					// remove aiff file
					os.Remove(fileName)
				} else {
					fmt.Printf("Error converting to mp3: %v\n", err)
				}

			}
			return
		default:

		}
	}

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
