package wsroom

import (
	"time"

	"github.com/gorilla/websocket"
)

// Connection ------------------------------------------------------------

type Connection struct {
	Key 	string

	Data 	map[interface{}]interface{}

	Conn 	*websocket.Conn

	Send	chan Message

	room 	Room
}

func (conn Connection) listen() {
	go conn.writePump()
	go conn.readPump()
}

func (conn Connection) writePump() {
	ticker := time.NewTicker(conn.room.PingPeriod)
	defer func(){
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(conn.room.WriteWait))

			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := conn.Conn.WriteJSON(msg)
			if err != nil {
				return
			}

		case <- ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(conn.room.WriteWait))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (conn Connection) readPump() {
	defer func() {
		conn.room.CommunicationChannels.UnRegiser <- conn
		conn.Conn.Close()
	}()
	
	conn.Conn.SetReadLimit(conn.room.MaxMessageSize)
	conn.Conn.SetReadDeadline(time.Now().Add(conn.room.PongWait))
	conn.Conn.SetPongHandler(func(appData string) error { conn.Conn.SetReadDeadline(time.Now().Add(conn.room.PongWait)); return nil })
	
	for {
		msg := conn.room.UnmarshalIn

		err := conn.Conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		conn.room.CommunicationChannels.Broadcast <- msg
	}
}

// CommunicationChannels --------------------------------------------------

type CommunicationChannels struct {
	Broadcast		chan Message

	Register		chan Connection

	UnRegiser		chan Connection
}


// Message ----------------------------------------------------------------

type Message interface {
	GetData() map[string]interface{}
}

type PlainMessage struct {
	Data	map[string]interface{}		`json:"data"`
}

func (m PlainMessage) GetData() map[string]interface{} { return m.Data }