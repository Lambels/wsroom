package wsroom

import (
	"errors"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomAlreadyExists = errors.New("room already exists")
)

// Store ----------------------------------------------------------------------

type Store interface {
	Get(key string) (Room, error)

	Delete(key string) error

	New(key string, maxMessageSize int64) (Room, error)
}

// RuntimeStore ----------------------------------------------------------------

func NewRuntimeStore() Store {
	return &RuntimeStore {
		Rooms: make(map[string]Room),
	}
}

type RuntimeStore struct {
	Rooms map[string]Room
}

// Get the room with the specific key
func (s *RuntimeStore) Get(key string) (Room, error) {
	room, ok := s.Rooms[key]
	if !ok {
		return room, ErrRoomNotFound
	}

	return room, nil
}

// Delete the room with the specific key
func (s *RuntimeStore) Delete(key string) error {
	room, ok := s.Rooms[key]
	if !ok {
		return ErrRoomNotFound
	}

	delete(s.Rooms, key)
	return room.Close()
}

// Create a room with the set key
func (s *RuntimeStore) New(key string, maxMessageSize int64) (Room, error) {
	if room, ok := s.Rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	room := NewRoom(key, maxMessageSize)
	s.Rooms[key] = room

	return room, nil
}