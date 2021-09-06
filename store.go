package wsroom

import (
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
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

func (s *RuntimeStore) Get(key string) (Room, error) {
	room, ok := s.Rooms[key]
	if !ok {
		return room, ErrRoomNotFound
	}

	return room, nil
}

func (s *RuntimeStore) Delete(key string) error {
	room, ok := s.Rooms[key]
	if !ok {
		return ErrRoomNotFound
	}

	delete(s.Rooms, key)
	return room.Close()
}

func (s *RuntimeStore) New(key string, maxMessageSize int64) (Room, error) {
	if room, ok := s.Rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	room := NewRoom(key, maxMessageSize)
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

	createTQ := "CREATE TABLE IF NOT EXISTS " +
		tableName + " (roomKey VARCHAR(100) NOT NULL, " +
		"ping_period INT, " +
		"write_wait INT, " +
		"pong_wait INT, " +
		"max_message_size INT, PRIMARY KEY(`roomKey`))"

	if _, err := db.Exec(createTQ); err != nil {		// If table already exits query is going to NOP
		return nil, err
	}

	insertQ := "INSERT INTO " + tableName + 
		" (roomKey, ping_period, write_wait, pong_wait, max_message_size) VALUES (?, ?, ?, ?, ?)"

	insertStmt, err := db.Prepare(insertQ)
	if err != nil {
		return nil, err
	}

	deleteQ := "DELETE FROM " + tableName + " WHERE roomKey = ?"

	deleteStmt, err := db.Prepare(deleteQ)
	if err != nil {
		return nil, err
	}

	selectQ := "SELECT * FROM " + tableName + " WHERE roomKey = ?"

	selectStmt, err := db.Prepare(selectQ)
	if err != nil {
		return nil, err
	}

	return &MySqlStore {
		Db: 				db,
		roomCache: 			cache.New(time.Minute * 5, time.Minute * 10),

		insertStmt: 		insertStmt,
		deleteStmt: 		deleteStmt,
		selectStmt: 		selectStmt,
	}, nil

}

type MySqlStore struct {
	Db					*sql.DB
	roomCache			*cache.Cache

	insertStmt			*sql.Stmt
	deleteStmt			*sql.Stmt
	selectStmt			*sql.Stmt
}

type roomColumn struct {
	key				string
	pingPeriod		time.Duration
	writeWait		time.Duration
	pongWait		time.Duration
	maxMessageSize	int64
}

func (s *MySqlStore) Get(key string) (Room, error) {
	var column roomColumn
	var room Room

	if res, exists := s.roomCache.Get(key); exists {	// Check the cache for room
		return res.(Room), nil
	}

	res, err := s.selectStmt.Query(key)
	switch err {
	case nil:

	case sql.ErrNoRows:
		return room, ErrRoomNotFound

	default:
		return room, err
	}

	err = res.Scan(&column)
	if err != nil {
		return room, err
	}

	r := NewRoom(key, column.maxMessageSize)

	s.roomCache.Set(key, r, cache.NoExpiration)

	return r, nil
}

func (s *MySqlStore) Delete(key string) error {
	var closeErr 	error

	if res, exists := s.roomCache.Get(key); exists {
		closeErr = res.(Room).Close()
	}

	_, err := s.deleteStmt.Exec(key)
	if err != nil {
		return err
	}

	return closeErr
}

func (s *MySqlStore) New(key string, maxMessageSize int64) (Room, error) {
	var room Room

	if res, exists := s.roomCache.Get(key); exists {
		return res.(Room), ErrRoomAlreadyExists
	}

	_, err := s.insertStmt.Exec(key, RegularPingPeriod, RegularWriteWait, RegularPongWait, maxMessageSize)
	switch err.(type) {
	case *mysql.MySQLError:
		if err.(*mysql.MySQLError).Number == 1062 {
			return room, ErrRoomAlreadyExists
		}
		return room, err

	case nil:

	default:
		return room, err
	}

	room = NewRoom(key, maxMessageSize)
	s.roomCache.Set(key, room, cache.NoExpiration)

	return room, nil
}