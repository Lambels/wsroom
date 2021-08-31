package wsroom

import (
	"database/sql"
	"errors"
	"github.com/patrickmn/go-cache"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomAlreadyExists = errors.New("room already exists")
)

// Store ----------------------------------------------------------------------

type Store interface {
	Get(key string) (Room, error)

	Delete(key string) error

	New(key string, maxMessageSize int64, messageStruct interface{}) (Room, error)
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

func (s RuntimeStore) New(key string, maxMessageSize int64, unmarshalIn interface{}) (Room, error) {
	if room, ok := s.Rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	room := NewRoom(key, unmarshalIn, maxMessageSize)
	s.Rooms[key] = room

	go room.listen()

	return room, nil
}

// MySqlStore ----------------------------------------------------------------

func NewMySqlStore(dsn, tableName string) (Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	_, err := db.Exec()
}

type MySqlStore struct {
	Db					*sql.DB
	tableName			string
	roomCache			*cache.Cache

	insertStmt			*sql.Stmt
	deleteStmt			*sql.Stmt
	selectStmt			*sql.Stmt
}