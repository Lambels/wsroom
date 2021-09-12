package main

import (
	"html/template"
	"net/http"

	"github.com/Lambels/wsroom"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	store = wsroom.NewRuntimeStore()	// Create our room store
	tpl *template.Template
	upgrader = websocket.Upgrader{}
)

// parse templates
func init() {
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
}

func main() {
	http.HandleFunc("/room/create", CreateRoom)
	http.HandleFunc("/ws", ConnectWS)
	http.HandleFunc("/room/join", RenderRoom)
	http.HandleFunc("/favicon.ico", http.NotFound)

	http.ListenAndServe(":8080", nil)
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		roomKey := r.FormValue("roomKey")
		
		// create a new room with the given roomKey and setting the message structure for this room
		_, err := store.New(roomKey, wsroom.RegularMaxMessageSize)
		// room already exists?
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	tpl.ExecuteTemplate(w, "createRoom.html", nil)
}
 
func ConnectWS(w http.ResponseWriter, r *http.Request) {
	roomKey, ok := r.URL.Query()["roomKey"]

	if !ok || len(roomKey[0]) < 1 {
		http.Error(w, "roomKey not found", http.StatusBadRequest)
		return
	}

	// get the room which coresponds to the given roomKey
	room, err := store.Get(roomKey[0])
	// room not found?
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// create a connection
	// each connection in a room must have an unique Key attribute
	conn := wsroom.Connection {
		Key: uuid.NewString(),
		Data: make(map[interface{}]interface{}),
		Conn: ws,
		Send: make(chan interface{}),
	}

	// subscribe the connection to the room
	err = room.Subscribe(conn)
	//! after upgrading the connection to a ws one the response writer gets hijaked by the upgrader
	//! so we can write to it anymore, any error will be written through the websocket connection
	if err != nil {
		ws.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
	}
}

func RenderRoom(w http.ResponseWriter, r *http.Request) {
	roomKey, ok := r.URL.Query()["roomKey"]

	if !ok || len(roomKey[0]) < 1 {
		http.Error(w, "RoomKey not found", http.StatusBadRequest)
		return
	}

	room, err := store.Get(roomKey[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// render the template with the room as data so we can access the roomKey
	tpl.ExecuteTemplate(w, "room.html", room)
}