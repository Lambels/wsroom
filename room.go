package wsroom

import (
	"log"
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

func NewRoom(key string, maxMessageSize int64) Room {
	return Room {
		Key: 						key,	
		Connections: 				make(map[string]Connection),
		PingPeriod: 				RegularPingPeriod,
		WriteWait: 					RegularWriteWait,
		PongWait: 					RegularPongWait,
		MaxMessageSize: 			maxMessageSize,
		CommunicationChannels: 		CommunicationChannels{
			Broadcast: 		make(chan interface{}),
			Register: 		make(chan Connection),
			UnRegiser: 		make(chan Connection),
		},
	}
}

// Room -------------------------------------------------

type Room struct {
	Key				string

	Connections 	map[string]Connection

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

func (r Room) Subscribe(conn Connection) (error) {
	if _, ok := r.Connections[conn.Key]; ok {
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

func (r Room) Broadcast(msg interface{}) {
	log.Println(r.Connections)
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