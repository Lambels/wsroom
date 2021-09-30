package wsroom

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// Connection ------------------------------------------------------------

type Connection struct {
	// Primary key used to identify each connection in a room
	Key string

	// The data saved on each connection
	Data map[interface{}]interface{}

	// The websocket connection used to communicated back and forth
	Conn *websocket.Conn

	// Messages which get sent to the Conn are taken from this channel
	Send chan interface{}

	// The room in which this connection is in
	room *Room
}

// listen starts the write and read pump
func (conn Connection) listen() {
	go conn.readPump()
	go conn.writePump()
}

// writePump is ran per connection and pumps messages from conn.Send
// to the conn.Conn and pings the connection to identify a dead connection
func (conn Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := conn.Conn.WriteJSON(msg)
			if err != nil {
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump is ran per connection and pumps messages from
// conn.Conn to the Broadcast channel
func (conn Connection) readPump() {
	defer func() {
		conn.room.CommunicationChannels.UnRegiser <- conn.Key
		conn.Conn.Close()
	}()

	conn.Conn.SetReadLimit(conn.room.maxMessageSize)
	conn.Conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.Conn.SetPongHandler(func(appData string) error { conn.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var msg map[string]interface{}

		err := conn.Conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		conn.room.CommunicationChannels.Broadcast <- msg
	}
}

// CommunicationChannels --------------------------------------------------

type CommunicationChannels struct {
	// Channel used for sending messages to all the connection
	// in the room, including the sender
	Broadcast chan interface{}

	// Channel used to register a connection
	Register chan Connection

	// Channel used to unregister a connection
	UnRegiser chan string
}
