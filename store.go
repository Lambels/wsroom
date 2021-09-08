package wsroom

import (
	"database/sql"
	"errors"
	"log"

	"github.com/go-sql-driver/mysql"
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
		"max_message_size INT, PRIMARY KEY(`roomKey`))"

	if _, err := db.Exec(createTQ); err != nil {		// If table already exits query is going to NOP
		return nil, err
	}

	insertQ := "INSERT INTO " + tableName + 
		" (roomKey, max_message_size) VALUES (?, ?)"

	insertStmt, err := db.Prepare(insertQ)
	if err != nil {
		log.Println("insert")
		return nil, err
	}

	deleteQ := "DELETE FROM " + tableName + " WHERE roomKey = ?"

	deleteStmt, err := db.Prepare(deleteQ)
	if err != nil {
		log.Println("delete")
		return nil, err
	}

	selectQ := "SELECT * FROM " + tableName + " WHERE roomKey = ?"

	selectStmt, err := db.Prepare(selectQ)
	if err != nil {
		log.Println("select")
		return nil, err
	}

	return &MySqlStore {
		Db: 				db,
		roomCache: 			make(map[string]Room),

		insertStmt: 		insertStmt,
		deleteStmt: 		deleteStmt,
		selectStmt: 		selectStmt,
	}, nil

}

type MySqlStore struct {
	Db					*sql.DB
	roomCache			map[string]Room

	insertStmt			*sql.Stmt
	deleteStmt			*sql.Stmt
	selectStmt			*sql.Stmt
}

type roomColumn struct {
	key				string
	maxMessageSize	int64
}

func (s *MySqlStore) Get(key string) (Room, error) {
	var col roomColumn
	var room Room

	if room, exists := s.roomCache[key]; exists {	// Check the cache for room
		log.Println("Room found in chache", room)
		return room, nil
	}

	rows, err := s.selectStmt.Query(key)
	switch err {
	case nil:

	case sql.ErrNoRows:
		log.Panicln("Room not found")
		return room, ErrRoomNotFound

	default:
		log.Println("Get err", err)
		return room, err
	}

	for rows.Next() {
		if err := rows.Scan(&col.key, &col.maxMessageSize); err != nil {
			return room, err
		}
		
		if err = rows.Err(); err != nil {
			return room, err
		}
	}

	r := NewRoom(key, col.maxMessageSize)

	s.roomCache[key] = r

	return r, nil
}

func (s *MySqlStore) Delete(key string) error {
	var closeErr 	error

	if room, exists := s.roomCache[key]; exists {
		closeErr = room.Close()
	}

	_, err := s.deleteStmt.Exec(key)
	if err != nil {
		return err
	}

	return closeErr
}

func (s *MySqlStore) New(key string, maxMessageSize int64) (Room, error) {
	var room Room

	if room, exists := s.roomCache[key]; exists {
		go room.listen()
		return room, ErrRoomAlreadyExists
	}

	_, err := s.insertStmt.Exec(key, maxMessageSize)
	switch err.(type) {
	case *mysql.MySQLError:
		if err.(*mysql.MySQLError).Number == 1062 {
			go room.listen()
			return room, ErrRoomAlreadyExists
		}
		return room, err

	case nil:

	default:
		return room, err
	}

	room = NewRoom(key, maxMessageSize)
	log.Println(room)

	s.roomCache[key] = room

	go room.listen()

	return room, nil
}