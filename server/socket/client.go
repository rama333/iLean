package socket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	typeClient   string
	serialNumber int
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	//c.conn.SetReadLimit(maxMessageSize)
	//c.conn.SetReadDeadline(time.Now().Add(pongWait))
	//c.conn.SetPongHandler(func(string) error {
	//	logrus.Info("pong")
	//	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	//	return ni
	//	l
	//})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var DataCommand struct {
			Command int         `json:"command,omitempty"`
			Data    interface{} `json:"data,omitempty"`
		}

		var DataCommand1 struct {
			Zone        int32   `json:"zone"`
			TempAir     float32 `json:"temp_air"`
			HumidityAir int `json:"humidity_air"`
			Tempfloor   float32 `json:"tempfloor"`
			CO2         float32 `json:"co_2"`
		}

		type DataCommand2 struct {
			Zone              int32   `json:"zone"`
			SetpointValueTemp float32 `json:"setpoint_value_temp"`
			TypeRegulation    int8    `json:"type_regulation"`
		}

		err = json.Unmarshal(message, &DataCommand1)

		if err == nil {
			DataCommand.Command = 1
			DataCommand.Data = DataCommand1
		}

		if err != nil {
			var data []DataCommand2
			err = json.Unmarshal(message, &data)
			if err != nil {
				logrus.Error(err)
				continue
			}

			DataCommand.Command = 2
			DataCommand.Data = data
		}

		logrus.Info(c.serialNumber)
		logrus.Info(c.typeClient)
		logrus.Info(DataCommand)

		data, err := json.Marshal(DataCommand)

		if err != nil {
			logrus.Error(err)
			continue
		}

		c.hub.Send("mobile", c.serialNumber, data)

		//message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		//
		//logrus.Info(message)

		//c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	//ticker := time.NewTicker(pingPeriod)
	//defer func() {
	//	ticker.Stop()
	//	c.conn.Close()
	//}()
	//for {
	//	select {
	//	case message, ok := <-c.send:
	//		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	//		if !ok {
	//			// The hub closed the channel.
	//			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	//			return
	//		}
	//
	//		w, err := c.conn.NextWriter(websocket.TextMessage)
	//		if err != nil {
	//			return
	//		}
	//		w.Write(message)
	//
	//		// Add queued chat messages to the current websocket message.
	//		n := len(c.send)
	//		for i := 0; i < n; i++ {
	//			w.Write(newline)
	//			w.Write(<-c.send)
	//		}
	//
	//		if err := w.Close(); err != nil {
	//			return
	//		}
	//	case <-ticker.C:
	//		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	//		if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
	//			return
	//		}
	//	}
	//}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	logrus.Info("new client (client pack)")

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	var greet struct {
		SerialNumber int    `json:"serial_number"`
		TypeClient   string `json:"type_client"`
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
		return
	}

	err = json.Unmarshal(message, &greet)

	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
		return
	}

	if greet.SerialNumber < 1 && (greet.TypeClient != "controller" || greet.TypeClient != "mobile") {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), serialNumber: greet.SerialNumber, typeClient: greet.TypeClient}

	client.hub.register <- client

	logrus.Info("okkkkkk")

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	err = conn.WriteMessage(websocket.TextMessage, []byte("ok"))

	if err != nil {
		logrus.Error(err)
	}
}
