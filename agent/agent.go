package agent

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/jacobsa/go-serial/serial"
	"github.com/sirupsen/logrus"
	"iLean/config"
	"iLean/entity"
	"io"
	"sync"
	"time"
)

const (
	preamOneByte byte = 85
	preamTwoByte byte = 170
)

type Agent struct {
	connect *websocket.Conn
	buffer  []byte
	port    io.ReadWriteCloser
	mutex   sync.Mutex
	Config  config.Config
	err     chan error
}

func NewAgent(conf *config.Config) (*Agent, error) {

	agent := new(Agent)
	agent.Config = *conf
	agent.mutex = sync.Mutex{}
	agent.err = make(chan error)

	logrus.Info("connect to socket")
	agent.connectToSocket()

	logrus.Info("connect to device")
	agent.connectToDevice()

	go agent.Connector()

	return agent, nil

}

func (a *Agent) Connector() {
	for {
		<-a.err

		logrus.Info("lost connection")

		logrus.Info("reconnect")

		a.connectToSocket()

		a.connectToDevice()

		a.ProcessingStream()

	}
}

func (a *Agent) connectToSocket() {

	for {

		connect, _, err := websocket.DefaultDialer.Dial("ws://45.141.79.96:63240/ws", nil)

		if err != nil {
			logrus.WithError(err).Error("failed to connect to server")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue
		}



		type Greet struct {
			SerialNumber int    `json:"serial_number"`
			TypeClient   string `json:"type_client"`
		}

		greet := Greet{SerialNumber: a.Config.Serial, TypeClient: "controller"}

		byteGreet, err := json.Marshal(greet)

		if err != nil {
			logrus.WithError(err).Error("failed marshal json")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue
		}

		err = connect.WriteMessage(websocket.TextMessage, byteGreet)

		if err != nil {
			logrus.WithError(err).Error("failed marshal json")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue
		}

		_, mes, err := connect.ReadMessage()

		if err != nil {
			logrus.WithError(err).Error("failed read messages from server")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue
		}

		if string(mes) == "ok" {
			logrus.Info("success connect to server")
			a.connect = connect
			return
		} else {
			logrus.WithError(err).Error("failed messages from server")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue
		}

	}
}

func (a *Agent) connectToDevice() {

	for {

		options := serial.OpenOptions{
			PortName:        "/dev/ttyAMA0",
			DataBits:        8,
			BaudRate:        115200,
			StopBits:        1,
			MinimumReadSize: 2,
		}

		// Open the port.
		port, err := serial.Open(options)

		if err != nil {
			logrus.Error("failed connect to device")
			time.Sleep(time.Second * 4)
			logrus.Info("reconnect")
			continue

		} else {
			logrus.Info("success connect to device")
			a.port = port
			return
		}

	}

}

func (a *Agent) ProcessingStream() {

	logrus.Info("start processing data")

	go func(agent *Agent) {

		for {

			_, mes, err := agent.connect.ReadMessage()

			logrus.Info(string(mes))

			if err != nil {
				logrus.Error(err)
				a.err <- err
				return
			}

			buf := bytes.NewBuffer([]byte{})

			if err := binary.Write(buf, binary.LittleEndian, []byte{preamOneByte, preamTwoByte, preamOneByte, preamTwoByte}); err != nil {
				logrus.Error(err)
			}

			var lenDataBytes int16 = 14

			if err := binary.Write(buf, binary.LittleEndian, lenDataBytes); err != nil {
				logrus.Error(err)
			}

			var addres int32 = 101

			if err := binary.Write(buf, binary.LittleEndian, addres); err != nil {
				logrus.Error(err)
			}

			var command int8 = 2

			if err := binary.Write(buf, binary.LittleEndian, command); err != nil {
				logrus.Error(err)
			}

			var commandTemperature entity.CommandTemperature
			var commandTemperatureBySensor entity.CommandTemperatureBySensor

			err = json.Unmarshal(mes, &commandTemperature)

			if err == nil && commandTemperature.Temperature > 0 {

				var zone4Byte int32 = int32(commandTemperature.Zone)

				if err := binary.Write(buf, binary.LittleEndian, zone4Byte); err != nil {
					logrus.Error(err)
				}

				var temperature4Byte float32 = float32(commandTemperature.Temperature)

				if err := binary.Write(buf, binary.LittleEndian, temperature4Byte); err != nil {
					logrus.Error(err)
				}

				var types uint8 = uint8(0)

				if err := binary.Write(buf, binary.LittleEndian, types); err != nil {
					logrus.Error(err)
				}

			}

			err = json.Unmarshal(mes, &commandTemperatureBySensor)

			if err == nil && commandTemperatureBySensor.Type > 0 {

				var zone4Byte int32 = int32(commandTemperatureBySensor.Zone)
				if err := binary.Write(buf, binary.LittleEndian, zone4Byte); err != nil {
					logrus.Error(err)
				}

				var temperature4Byte float32 = float32(0)

				if err := binary.Write(buf, binary.LittleEndian, temperature4Byte); err != nil {
					logrus.Error(err)
				}

				var types uint8 = uint8(commandTemperatureBySensor.Type)

				if err := binary.Write(buf, binary.LittleEndian, types); err != nil {
					logrus.Error(err)
				}

			}

			var h uint16 = uint16(32767)

			if err := binary.Write(buf, binary.LittleEndian, h); err != nil {
				logrus.Error(err)
			}

			logrus.Info("--------------------------------")
			logrus.Info(buf.Bytes())
			logrus.Info("--------------------------------")

			t := time.Now()
			_, err = a.port.Write(buf.Bytes())
			if err != nil {
				logrus.WithError(err).Error("failed write to device")
			}

			logrus.Info(time.Now().Sub(t).Seconds(), " sec")
			logrus.Info(time.Now().Sub(t).Milliseconds(), " milli")
			logrus.Info(time.Now().Sub(t).Nanoseconds(), " nano")

			logrus.Info("--------------------------------")

		}
	}(a)

	//go func(agent *Agent) {
	//
	//	for {
	//		agent.connect.SetWriteDeadline(time.Now().Add(60 * time.Second))
	//
	//		agent.mutex.Lock()
	//		if err := agent.connect.WriteMessage(websocket.PingMessage, nil); err != nil {
	//			logrus.Error(err)
	//			a.err <- err
	//			return
	//		}
	//		agent.mutex.Unlock()
	//
	//		agent.connect.SetPingHandler(func(string) error {
	//			logrus.Info("ping")
	//			agent.connect.SetReadDeadline(time.Now().Add(60 * time.Second))
	//			return nil
	//		})
	//
	//		time.Sleep(20 * time.Second)
	//	}
	//
	//}(a)

	preamOne := false
	preamTwo := false
	preamThree := false
	preamFour := false

	for {

		//time.Sleep(1 * time.Second)

		buf := make([]byte, 1)
		//buf := a.readOneByte(4)

		//logrus.Info("start read stream bytes")
		_, err := a.port.Read(buf)

		if err != nil {
			if err != io.EOF {
				logrus.Println("Error reading from serial port: ", err)
			}

		} else {

			if buf[0] == preamOneByte || preamOne {
				preamOne = true

				if buf[0] == preamOneByte && !preamTwo {
					continue
				}

				if buf[0] == preamTwoByte || preamTwo {
					preamTwo = true

					if buf[0] == preamTwoByte && !preamThree {
						continue
					}

					if buf[0] == preamOneByte || preamThree {
						preamThree = true

						if buf[0] == preamOneByte {
							continue
						}

						if buf[0] == preamTwoByte || preamFour {
							preamFour = true

						} else {
							preamOne = false
							preamTwo = false
							preamThree = false
							preamFour = false
						}
					} else {
						preamOne = false
						preamTwo = false
						preamThree = false
						preamFour = false
					}
				} else {
					preamOne = false
					preamTwo = false
					preamThree = false
					preamFour = false
				}

			} else {
				preamOne = false
				preamTwo = false
				preamThree = false
				preamFour = false
			}

			if preamOne && preamTwo && preamThree && preamFour {

				preamOne = false
				preamTwo = false
				preamThree = false
				preamFour = false

				lenDataBuf := make([]byte, 2)
				_, err := a.port.Read(lenDataBuf)
				if err != nil {
					if err != io.EOF {
						logrus.Println("Error reading from serial port: ", err)
					}
				}

				var lenDataBytes int16

				buf := bytes.NewReader(lenDataBuf)
				err = binary.Read(buf, binary.LittleEndian, &lenDataBytes)
				if err != nil {
					logrus.Error("binary.Read failed:", err)
				}

				data := make([]byte, 5)
				_, err = a.port.Read(data)
				if err != nil {
					if err != io.EOF {
						logrus.Println("Error reading from serial port: ", err)
					}
				}

				type DataAddresCommand struct {
					Addres  int32 `json:"addres"`
					Command int8  `json:"command"`
				}

				var dataAddresCommand DataAddresCommand
				bufReader := bytes.NewReader(data)
				err = binary.Read(bufReader, binary.LittleEndian, &dataAddresCommand)
				if err != nil {
					logrus.Error("binary.Read failed:", err)
					logrus.Info("datebase", dataAddresCommand)
				}

				if dataAddresCommand.Command == 1 {

					logBuff := make([]byte, 0)

					type Data struct {
						Zone              int32   `json:"zone"`
						SetpointValueTemp float32 `json:"setpoint_value_temp"`
						TypeRegulation    int8    `json:"type_regulation"`
					}

					dataSlice := make([]Data, 0)

					var crc uint16

					for i := 0; i < 6; i++ {

						var zone int32
						var setpointValueTemp float32
						var typeRegulation int8

						{
							data = a.readOneByte(4)

							buf = bytes.NewReader(data)
							err = binary.Read(buf, binary.LittleEndian, &zone)
							if err != nil {
								logrus.Error("binary.Read failed:", err)
							}

							for _, item := range data {
								logBuff = append(logBuff, item)
							}

						}

						{
							data = a.readOneByte(4)

							buf = bytes.NewReader(data)
							err = binary.Read(buf, binary.LittleEndian, &setpointValueTemp)
							if err != nil {
								logrus.Error("binary.Read failed:", err)
							}

							for _, item := range data {
								logBuff = append(logBuff, item)
							}

						}

						{
							data = a.readOneByte(1)

							buf = bytes.NewReader(data)
							err = binary.Read(buf, binary.LittleEndian, &typeRegulation)
							if err != nil {
								logrus.Error("binary.Read failed:", err)
							}
							for _, item := range data {
								logBuff = append(logBuff, item)
							}
							//logrus.Info("read in ", n, " bytes")

							typeRegulation -= 1
						}

						dataSlice = append(dataSlice, Data{
							zone,
							setpointValueTemp,
							typeRegulation,
						})

					}

					{
						data = a.readOneByte(2)

						buf = bytes.NewReader(data)
						err = binary.Read(buf, binary.LittleEndian, &crc)
						if err != nil {
							logrus.Error("binary.Read failed:", err)
						}

						//logrus.Info("read crc in ", n, " bytes")

						for _, item := range data {
							logBuff = append(logBuff, item)
						}
					}

					a.readOneByte(1)

					dataBytes, err := json.Marshal(dataSlice)

					if err != nil {
						logrus.Error(err)
					}

					a.mutex.Lock()
					err = a.connect.WriteMessage(websocket.TextMessage, dataBytes)
					a.mutex.Unlock()

					if err != nil {
						logrus.WithError(err).Error("connection to lost")
						a.err <- err
						return
					}

				}

				if dataAddresCommand.Command == 0 {

					var tempAir float32
					var humidityAir float32
					var tempfloor float32
					var co2 float32
					var crc uint16

					{
						//data = make([]byte, 4)
						data = a.readOneByte(4)
						//_, err = a.port.Read(data)
						//if err != nil {
						//	if err != io.EOF {
						//		logrus.Println("Error reading from serial port: ", err)
						//	}
						//}

						buf := bytes.NewReader(data)
						err = binary.Read(buf, binary.LittleEndian, &tempAir)
						if err != nil {
							logrus.Error("binary.Read failed:", err)
						}
					}

					{
						//data = make([]byte, 4)
						data = a.readOneByte(4)
						//_, err = a.port.Read(data)
						//if err != nil {
						//	if err != io.EOF {
						//		logrus.Println("Error reading from serial port: ", err)
						//	}
						//}

						buf = bytes.NewReader(data)
						err = binary.Read(buf, binary.LittleEndian, &humidityAir)
						if err != nil {
							logrus.Error("binary.Read failed:", err)
						}
					}

					{
						//data = make([]byte, 4)
						data = a.readOneByte(4)
						//_, err = a.port.Read(data)
						//if err != nil {
						//	if err != io.EOF {
						//		logrus.Println("Error reading from serial port: ", err)
						//	}
						//}
						buf = bytes.NewReader(data)
						err = binary.Read(buf, binary.LittleEndian, &tempfloor)
						if err != nil {
							logrus.Error("binary.Read failed:", err)
						}

						//logrus.Info("read in ", n, " bytes")
					}

					{
						//data = make([]byte, 2)
						data = a.readOneByte(2)
						//_, err = a.port.Read(data)
						//if err != nil {
						//	if err != io.EOF {
						//		logrus.Println("Error reading from serial port: ", err)
						//	}
						//}
						buf = bytes.NewReader(data)
						err = binary.Read(buf, binary.LittleEndian, &crc)
						if err != nil {
							logrus.Error("binary.Read failed:", err)
						}

						//	logrus.Info("read crc in ", n, " bytes")
					}

					//{
					//	data = make([]byte, 4)
					//	_, err = a.port.Read(data)
					//	if err != nil {
					//		if err != io.EOF {
					//			logrus.Println("Error reading from serial port: ", err)
					//		}
					//	}
					//	buf = bytes.NewReader(data)
					//	err = binary.Read(buf, binary.LittleEndian, &co2)
					//	if err != nil {
					//		logrus.Error("binary.Read failed:", err)
					//	}
					//}

					type DataCommandZero struct {
						Zone        int32   `json:"zone"`
						TempAir     float32 `json:"temp_air"`
						HumidityAir int     `json:"humidity_air"`
						Tempfloor   float32 `json:"tempfloor"`
						CO2         float32 `json:"co_2"`
					}

					dataCommandZero := DataCommandZero{
						dataAddresCommand.Addres,
						tempAir,
						int(humidityAir),
						tempfloor,
						co2,
					}

					dataBytes, err := json.Marshal(dataCommandZero)

					if err != nil {
						logrus.Error(err)
					}

					a.mutex.Lock()
					err = a.connect.WriteMessage(websocket.TextMessage, dataBytes)
					a.mutex.Unlock()

					if err != nil {
						logrus.WithError(err).Error("connection to lost")
						a.err <- err
						return
					}

					//buf := bytes.NewReader(data[5:21])
					//err = binary.Read(buf, binary.LittleEndian, &dataCommandZero)
					//if err != nil {
					//	logrus.Error("binary.Read failed:", err)
					//}
					//
					//logrus.Info(dataCommandZero)

					//var ResponseMessageCommandZero struct{
					//	DataCommandZero
					//
					//
					//}
					//
					//dataBytes, err := json.Marshal(Data)
					//
					//if err != nil {
					//	logrus.Error(err)
					//}

					//err = a.connect.WriteMessage(websocket.TextMessage, dataBytes)
					//
					//if err != nil {
					//	logrus.Error(err)
					//}
					//
					//logrus.Info("Data", Data)
				}

				//if lenDataBytes-5%9 == 0 {
				//
				//	dataBufBase := make([]byte, 5)
				//	_, err := a.port.Read(dataBufBase)
				//	if err != nil {
				//		if err != io.EOF {
				//			logrus.Println("Error reading from serial port: ", err)
				//		}
				//	}
				//
				//	type DataBase struct {
				//		Addres  int32 `json:"addres"`
				//		Command int8  `json:"command"`
				//	}
				//
				//	var dataBase DataBase
				//	buf := bytes.NewReader(dataBufBase)
				//	err = binary.Read(buf, binary.LittleEndian, &dataBase)
				//	if err != nil {
				//		logrus.Error("binary.Read failed:", err)
				//	}
				//
				//	logrus.Info("Data base", dataBase)
				//
				//	dataBuf := make([]byte, lenDataBytes)
				//	_, err = a.port.Read(dataBuf)
				//	if err != nil {
				//		if err != io.EOF {
				//			logrus.Println("Error reading from serial port: ", err)
				//		}
				//	}
				//
				//	type Data struct {
				//		Zone              int32   `json:"zone"`
				//		SetpointValueTemp float32 `json:"setpoint_value_temp"`
				//		TypeRegulation    int8    `json:"type_regulation"`
				//	}
				//
				//	dataSlice := make([]Data, 0)
				//
				//	firstIndex := 0
				//	lastIndex := 9
				//
				//	for i := 0; i < (int(lenDataBytes-5) / 9); i++ {
				//
				//		var data Data
				//		buf := bytes.NewReader(dataBuf[firstIndex:lastIndex])
				//		err = binary.Read(buf, binary.LittleEndian, &data)
				//		if err != nil {
				//			logrus.Error("binary.Read failed:", err)
				//		}
				//
				//		dataSlice = append(dataSlice, data)
				//
				//		firstIndex += 9
				//		lastIndex += 9
				//
				//	}
				//
				//	dataBytes, err := json.Marshal(dataSlice)
				//
				//	if err != nil {
				//		logrus.Error(err)
				//	}
				//
				//	err = a.connect.WriteMessage(websocket.TextMessage, dataBytes)
				//
				//	if err != nil {
				//		logrus.Error(err)
				//	}
				//
				//	logrus.Info("array ", dataSlice)
				//
				//}

				//
				//	log.Println("ok")
			}

			//buf = buf[:n]
			//fmt.Println(buf[:n])
			//fmt.Println("Rx: ", hex.EncodeToString(buf))
		}
		//              fmt.Println("Pause 1 second")
		//              time.Sleep(time.Second * 1)
	}
}

func (a *Agent) readOneByte(lenBuff int) []byte {

	//logrus.Info("start read")

	commonData := make([]byte, 0)

	for i := 0; i < lenBuff; i++ {
		data := make([]byte, 1)

		_, err := a.port.Read(data)
		if err != nil {
			if err != io.EOF {
				logrus.Println("Error reading from serial port: ", err)
			}
		}

		commonData = append(commonData, data[0])
	}

	return commonData
}

func (a *Agent) Close() {
	a.port.Close()
}

func test(b []byte, l int) {

	//t := [256]uint16{
	//0x0000, 0x1021, 0x2042, 0x3063, 0x4084, 0x50A5, 0x60C6, 0x70E7,
	//0x8108, 0x9129, 0xA14A, 0xB16B, 0xC18C, 0xD1AD, 0xE1CE, 0xF1EF,
	//0x1231, 0x0210, 0x3273, 0x2252, 0x52B5, 0x4294, 0x72F7, 0x62D6,
	//0x9339, 0x8318, 0xB37B, 0xA35A, 0xD3BD, 0xC39C, 0xF3FF, 0xE3DE,
	//0x2462, 0x3443, 0x0420, 0x1401, 0x64E6, 0x74C7, 0x44A4, 0x5485,
	//0xA56A, 0xB54B, 0x8528, 0x9509, 0xE5EE, 0xF5CF, 0xC5AC, 0xD58D,
	//0x3653, 0x2672, 0x1611, 0x0630, 0x76D7, 0x66F6, 0x5695, 0x46B4,
	//0xB75B, 0xA77A, 0x9719, 0x8738, 0xF7DF, 0xE7FE, 0xD79D, 0xC7BC,
	//0x48C4, 0x58E5, 0x6886, 0x78A7, 0x0840, 0x1861, 0x2802, 0x3823,
	//0xC9CC, 0xD9ED, 0xE98E, 0xF9AF, 0x8948, 0x9969, 0xA90A, 0xB92B,
	//0x5AF5, 0x4AD4, 0x7AB7, 0x6A96, 0x1A71, 0x0A50, 0x3A33, 0x2A12,
	//0xDBFD, 0xCBDC, 0xFBBF, 0xEB9E, 0x9B79, 0x8B58, 0xBB3B, 0xAB1A,
	//0x6CA6, 0x7C87, 0x4CE4, 0x5CC5, 0x2C22, 0x3C03, 0x0C60, 0x1C41,
	//0xEDAE, 0xFD8F, 0xCDEC, 0xDDCD, 0xAD2A, 0xBD0B, 0x8D68, 0x9D49,
	//0x7E97, 0x6EB6, 0x5ED5, 0x4EF4, 0x3E13, 0x2E32, 0x1E51, 0x0E70,
	//0xFF9F, 0xEFBE, 0xDFDD, 0xCFFC, 0xBF1B, 0xAF3A, 0x9F59, 0x8F78,
	//0x9188, 0x81A9, 0xB1CA, 0xA1EB, 0xD10C, 0xC12D, 0xF14E, 0xE16F,
	//0x1080, 0x00A1, 0x30C2, 0x20E3, 0x5004, 0x4025, 0x7046, 0x6067,
	//0x83B9, 0x9398, 0xA3FB, 0xB3DA, 0xC33D, 0xD31C, 0xE37F, 0xF35E,
	//0x02B1, 0x1290, 0x22F3, 0x32D2, 0x4235, 0x5214, 0x6277, 0x7256,
	//0xB5EA, 0xA5CB, 0x95A8, 0x8589, 0xF56E, 0xE54F, 0xD52C, 0xC50D,
	//0x34E2, 0x24C3, 0x14A0, 0x0481, 0x7466, 0x6447, 0x5424, 0x4405,
	//0xA7DB, 0xB7FA, 0x8799, 0x97B8, 0xE75F, 0xF77E, 0xC71D, 0xD73C,
	//0x26D3, 0x36F2, 0x0691, 0x16B0, 0x6657, 0x7676, 0x4615, 0x5634,
	//0xD94C, 0xC96D, 0xF90E, 0xE92F, 0x99C8, 0x89E9, 0xB98A, 0xA9AB,
	//0x5844, 0x4865, 0x7806, 0x6827, 0x18C0, 0x08E1, 0x3882, 0x28A3,
	//0xCB7D, 0xDB5C, 0xEB3F, 0xFB1E, 0x8BF9, 0x9BD8, 0xABBB, 0xBB9A,
	//0x4A75, 0x5A54, 0x6A37, 0x7A16, 0x0AF1, 0x1AD0, 0x2AB3, 0x3A92,
	//0xFD2E, 0xED0F, 0xDD6C, 0xCD4D, 0xBDAA, 0xAD8B, 0x9DE8, 0x8DC9,
	//0x7C26, 0x6C07, 0x5C64, 0x4C45, 0x3CA2, 0x2C83, 0x1CE0, 0x0CC1,
	//0xEF1F, 0xFF3E, 0xCF5D, 0xDF7C, 0xAF9B, 0xBFBA, 0x8FD9, 0x9FF8,
	//0x6E17, 0x7E36, 0x4E55, 0x5E74, 0x2E93, 0x3EB2, 0x0ED1, 0x1EF0,
	//}

	//var i uint16 = 0xFFFF
	//var crc16 uint16 = 0xFFFF
	//
	//for i := 0xFFFF; i < l; i++ {
	//	crc16 = (crc16 << 8) ^ t[(crc16 >> 8) ^ &b[0]]
	//}

}
