package wsroom

import (
	"time"

	"github.com/pkg/errors"
)

const (
	RegularWriteWait = time.Second * 10
	RegularPongWait = time.Second * 60
	RegularPingPeriod = (RegularPongWait * 9) / 10
	RegularMaxMessageSize = 512
)

var (
	ErrConnAlreadyExists = errors.New("connection already subscribed")
	ErrConnNotFound = errors.New("connection not found")
)

func NewRoom(unmarshalIn Message, pingPeriod, writeWait, pongWait time.Duration, maxMessageSize int64) Room {
	return Room {
		Connections: 				make(map[string]Connection),
		UnmarshalIn: 				unmarshalIn,
		PingPeriod: 				pingPeriod,
		WriteWait: 					writeWait,
		PongWait: 					pongWait,
		MaxMessageSize: 			maxMessageSize,
		CommunicationChannels: 		CommunicationChannels{
			Broadcast: 		make(chan Message),
			Register: 		make(chan Connection),
			UnRegiser: 		make(chan Connection),
		},
	}
}

// Room -------------------------------------------------

type Room struct {
	Connections 	map[string]Connection

	UnmarshalIn		Message

	PingPeriod 		time.Duration

	WriteWait		time.Duration

	PongWait		time.Duration

	MaxMessageSize	int64

	CommunicationChannels
}

func (r Room) Close() (error) {
	for _, conn := range r.Connections {
		close(conn.Send)
	}

	return nil
}

func (r Room) Subscribe(key string, conn Connection) (error) {
	if _, ok := r.Connections[key]; !ok {
		return ErrConnAlreadyExists
	}

	conn.room = r

	r.CommunicationChannels.Register <- conn
	return nil
}

func (r Room) UnSubscribe(key string) (error) {
	conn, ok := r.Connections[key]
	if !ok {
		return ErrConnNotFound
	}
	close(conn.Send)

	delete(r.Connections, key)
	return nil
}

func (r Room) Broadcast(msg Message) {
	for _, conn := range r.Connections {
		select {
		case conn.Send <- msg:
		
		default:
			r.CommunicationChannels.UnRegiser <- conn
		}

	}
}

func (r Room) listen() {
	for {
		select {
		case conn := <-r.CommunicationChannels.Register:
			r.Connections[conn.Key] = conn
			conn.listen()

		case conn := <-r.CommunicationChannels.UnRegiser:
			delete(r.Connections, conn.Key)
			close(conn.Send)
		
		case msg := <-r.CommunicationChannels.Broadcast:
			r.Broadcast(msg)
		}
	}
}