package wsroom

import (
	"errors"
	"time"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomAlreadyExists = errors.New("room already exists")
)

type Store interface {
	Get(key string) (Room, error)

	Delete(key string) error

	New(key string, pingPeriod, writeWait, pongWait time.Duration, maxMessageSize int64, unmarshalIn Message) (Room, error)
}

type RuntimeStore struct {
	Rooms map[string]Room
}

func (s RuntimeStore) Get(key string) (Room, error) {
	room, ok := s.Rooms[key]
	if !ok {
		return room, ErrRoomNotFound
	}

	return room, nil
}

func (s RuntimeStore) Delete(key string) error {
	room, ok := s.Rooms[key]
	if !ok {
		return ErrRoomNotFound
	}

	delete(s.Rooms, key)
	return room.Close()
}

func (s RuntimeStore) New(key string, pingPeriod, writeWait, pongWait time.Duration, maxMessageSize int64, unmarshalIn Message) (Room, error) {
	if room, ok := s.Rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	room := NewRoom(unmarshalIn, pingPeriod, writeWait, pongWait, maxMessageSize)
	s.Rooms[key] = room

	go room.listen()

	return room, nil
}