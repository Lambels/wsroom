package wsroom

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

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

	New(key string, maxMessageSize int64, closePeriod time.Duration) (Room, error)
}

// RuntimeStore ---------------------------------------------------------------

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
func (s *RuntimeStore) New(key string, maxMessageSize int64, closePeriod time.Duration) (Room, error) {
	if room, ok := s.Rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	room := NewRoom(key, maxMessageSize, closePeriod)
	s.Rooms[key] = room

	return room, nil
}

// DatabaseStore --------------------------------------------------------------

func NewSQLStore(driverName, dsn, tableName string) (Store, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	createTQ := fmt.Sprintf (
		`CREATE TABLE IF NOT EXISTS %s (
		roomKey VARCHAR(100) NOT NULL,
		max_message_size INT,
		close_period INT,
		PRIMARY KEY(roomKey))`, tableName)

	if _, err := db.Exec(createTQ); err != nil {	// statement will no-op if table already exists
		return nil, err
	}

	insertQ := fmt.Sprintf(`INSERT INTO %s (roomKey, max_message_size, close_period) VALUES (?, ?, ?)`, tableName)
	insertStmt, err := db.Prepare(insertQ)
	if err != nil {
		return nil, err
	}

	deleteQ := fmt.Sprintf(`DELETE FROM %s WHERE roomKey = ?`, tableName)
	deleteStmt, err := db.Prepare(deleteQ)
	if err != nil {
		return nil, err
	}

	selectQ := fmt.Sprintf(`SELECT * FROM %s WHERE roomKey = ?`, tableName)
	selectStmt, err := db.Prepare(selectQ)
	if err != nil {
		return nil, err
	}

	return &SQLStore {
		db: 			db,
		insertStmt: 	insertStmt,
		deleteStmt: 	deleteStmt,
		selectStmt: 	selectStmt,
		rooms: 	make(map[string]Room),
	}, nil
}

type SQLStore struct {
	db 			*sql.DB

	insertStmt 	*sql.Stmt
	deleteStmt	*sql.Stmt
	selectStmt	*sql.Stmt

	rooms	map[string]Room
}

type roomColumn struct {
	roomKey				string
	maxMessageSize		int64
	closePeriod			time.Duration
}

func (s *SQLStore) Get(key string) (Room, error) {
	var room 	Room
	var col 	roomColumn

	if room, ok := s.rooms[key]; ok {
		return room, nil
	}

	rows, err := s.selectStmt.Query(key)
	switch err {
	case nil:

	case sql.ErrNoRows:
		return room, ErrRoomNotFound

	default:
		return room, err
	}

	for rows.Next() {
		if err := rows.Scan(&col.roomKey, &col.maxMessageSize, &col.closePeriod); err != nil {
			return room, err
		}

		if err = rows.Err(); err != nil {
			return room, err
		}
	}


	r := NewRoom(key, col.maxMessageSize, col.closePeriod)
	s.rooms[key] = r

	return r, nil
} 

func (s *SQLStore) Delete(key string) error {
	var closeErr error

	if room, ok := s.rooms[key]; ok {
		closeErr = room.Close()
		delete(s.rooms, key)
	}

	_, err := s.deleteStmt.Exec(key)
	if err != nil {
		return err
	}

	return closeErr
}

func (s *SQLStore) New(key string, maxMessageSize int64, closePeriod time.Duration) (Room, error) {
	var room Room

	if room, ok := s.rooms[key]; ok {
		return room, ErrRoomAlreadyExists
	}

	_, err := s.insertStmt.Exec(key, maxMessageSize, closePeriod)
	switch err := err.(type) {
	case *mysql.MySQLError:
		if err.Number == 1062 {
			break
		}
		return room, err

	case nil:

	default:
		return room, err
	}

	room = NewRoom(key, maxMessageSize, closePeriod)
	s.rooms[key] = room

	return room, nil
}