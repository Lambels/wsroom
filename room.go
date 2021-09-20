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

// NewRoom is a helper function to create a Room using the default values.
func NewRoom(key string, maxMessageSize int64, closePeriod time.Duration) Room {
	r := Room {
		Key: 						key,	
		Connections: 				make(map[string]Connection),
		PingPeriod: 				RegularPingPeriod,
		WriteWait: 					RegularWriteWait,
		PongWait: 					RegularPongWait,
		MaxMessageSize: 			maxMessageSize,
		ClosePeriod: 				closePeriod,
		isListening: 				false,
		CommunicationChannels: 		CommunicationChannels{
			Broadcast: 		make(chan interface{}),
			Register: 		make(chan Connection),
			UnRegiser: 		make(chan Connection),
		},
	}

	return r
}

// Room -------------------------------------------------

type Room struct {
	// Primary key used to identify each room in a store
	Key				string

	// The active connections belonging to the room
	Connections 	map[string]Connection

	// The time between each ping message
	PingPeriod 		time.Duration

	// The time allowed to write a message to the room
	WriteWait		time.Duration

	// The time allowed to read the next pong message
	PongWait		time.Duration

	ClosePeriod		time.Duration

	// Maximum message size allowed
	MaxMessageSize	int64

	close			chan bool

	isListening		bool

	// The communication channels of the room
	CommunicationChannels
}

func (r Room) Close() (error) {
	r.close <- true
	return nil
}

// Close closes all the live connections to the room
func (r Room) CloseConnections() {
	for _, conn := range r.Connections {
		close(conn.Send)
	}
}

// Subscribe registers conn to the room called upon
func (r Room) Subscribe(conn Connection) (error) {
	if _, ok := r.Connections[conn.Key]; ok {
		return ErrConnAlreadyExists
	}

	if !r.isListening {
		go r.listen()
	}

	conn.room = r

	r.CommunicationChannels.Register <- conn
	return nil
}

// Unsubscribe unregisters a connection to the room called upon
func (r Room) UnSubscribe(key string) (error) {
	conn, ok := r.Connections[key]
	if !ok {
		return ErrConnNotFound
	}

	r.CommunicationChannels.UnRegiser <- conn
	return nil
}

// Broadcast puts msg into the Send channel of each connection
// and unregisters the connection if the channel isnt available (connection closed)
func (r Room) Broadcast(msg interface{}) {
	for _, conn := range r.Connections {
		select {
		case conn.Send <- msg:
		
		default:
			r.CommunicationChannels.UnRegiser <- conn
		}

	}
}

// listen starts listening on all the CommunicationChannels
func (r Room) listen() {
	ticker := time.NewTicker(r.ClosePeriod)
	defer func(){ ticker.Stop(); r.isListening = false }()

	r.close = make(chan bool, 1)
	r.isListening = true

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

		case <-ticker.C:
			if len(r.Connections) == 0 {
				return
			}

		case <-r.close:
			r.CloseConnections()
			return
		}
	}
}