package agent


import (
//      "encoding/hex"
"fmt"
"io"
"log"
//      "time"

"github.com/jacobsa/go-serial/serial"
)

func main() {
	// Set up options.
	options := serial.OpenOptions{
		PortName:        "/dev/ttyAMA0",
		DataBits:        8,
		BaudRate: 115200,
		StopBits:        1,
		MinimumReadSize: 2,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	defer port.Close()

	var preamOneByte byte =  85
	var preamTwoByte byte =  170


	for {
		buf := make([]byte, 4)
		_, err := port.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading from serial port: ", err)
			}
		} else {
			//                      log.Println(buf)
			if buf[0] == preamOneByte && buf[1] == preamTwoByte && buf[2] == preamOneByte && buf[3] == preamTwoByte {
				lenBuf := make([]byte, 2)
				_, err := port.Read(lenBuf)
				if err != nil {
					if err != io.EOF {
						fmt.Println("Error reading from serial port: ", err)
					}}
				log.Println(lenBuf)
				log.Println("ok")
			}

			//buf = buf[:n]
			//fmt.Println(buf[:n])
			//fmt.Println("Rx: ", hex.EncodeToString(buf))
		}
		//              fmt.Println("Pause 1 second")
		//              time.Sleep(time.Second * 1)
	}

}

