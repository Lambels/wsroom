package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/Lambels/wsroom"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	tpl 		*template.Template
	store = wsroom.NewRuntimeStore() // Create our room store
	upgrader = websocket.Upgrader{}
)

// parse templates
func init() {
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
}

func main() {
	// create the notifications room
	_, err := store.New("notifications", wsroom.RegularMaxMessageSize)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/notifications/ws", connectWS)
	http.HandleFunc("/create_notifiaction", createNotification)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/", index)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.html", nil)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	msg := map[string]interface{} {"foo": "bar"}

	// Get the notifications room
	room, err := store.Get("notifications")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// send the msg (notification) to the rooms Broadcast channel
	room.CommunicationChannels.Broadcast <- msg
}

func connectWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// create a connection
	// each connection in a room must have an unique Key attribute
	conn := wsroom.Connection {
		Key:	uuid.NewString(),
		Data: 	make(map[interface{}]interface{}),
		Conn: 	ws,
		Send: 	make(chan interface{}),
	}

	room, err := store.Get("notifications")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// subscribe the connection to the room
	err = room.Subscribe(conn)
	//! after upgrading the connection to a ws one the response writer gets hijaked by the upgrader
	//! so we can write to it anymore, any error will be written through the websocket connection
	if err != nil {
		ws.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
	}
}