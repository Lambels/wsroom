package wsroom

import (
	"time"

	"github.com/pkg/errors"
)

const (
	RegularMaxMessageSize = 512
)

var (
	ErrConnAlreadyExists = errors.New("connection already subscribed")
	ErrConnNotFound      = errors.New("connection not found")
)

// NewRoom is a helper function to create a Room using the default values.
func NewRoom(key string, maxMessageSize int64, closePeriod time.Duration) *Room {
	r := &Room{
		Key:            key,
		maxMessageSize: maxMessageSize,
		closePeriod:    closePeriod,
		CommunicationChannels: CommunicationChannels{
			Broadcast: make(chan interface{}),
			Register:  make(chan Connection),
			UnRegiser: make(chan string),
		},
	}

	return r
}

// Room -------------------------------------------------

type Room struct {
	// Primary key used to identify each room in a store
	Key string

	closePeriod time.Duration

	// Maximum message size allowed
	maxMessageSize int64

	close chan bool

	isListening bool

	// The communication channels of the room
	CommunicationChannels
}

func (r *Room) Close() error {
	r.close <- true
	return nil
}

// Close closes all the live connections to the room
func (r *Room) CloseConnections() {
	r.close <- true
}

// Subscribe registers conn to the room called upon
func (r *Room) Subscribe(conn Connection) error {
	if !r.isListening {
		go r.listen()
	}

	conn.room = r

	r.CommunicationChannels.Register <- conn
	return nil
}

// Unsubscribe unregisters a connection to the room called upon
func (r *Room) UnSubscribe(key string) {
	r.CommunicationChannels.UnRegiser <- key
}

// listen starts listening on all the CommunicationChannels
func (r *Room) listen() {
	var listeningOn map[string]Connection

	ticker := time.NewTicker(r.closePeriod)
	defer func() { ticker.Stop(); r.isListening = false }()

	listeningOn = make(map[string]Connection)
	r.close = make(chan bool, 1)
	r.isListening = true

	for {
		select {
		case conn := <-r.CommunicationChannels.Register:
			listeningOn[conn.Key] = conn
			conn.listen()

		case key := <-r.CommunicationChannels.UnRegiser:
			conn := listeningOn[key]
			delete(listeningOn, key)
			close(conn.Send)

		case msg := <-r.CommunicationChannels.Broadcast:
			for _, conn := range listeningOn {
				select {
				case conn.Send <- msg:

				default:
					r.CommunicationChannels.UnRegiser <- conn.Key
				}

			}

		case <-ticker.C:
			if len(listeningOn) == 0 {
				return
			}

		case <-r.close:
			for _, conn := range listeningOn {
				close(conn.Send)
			}
			return
		}
	}
}
