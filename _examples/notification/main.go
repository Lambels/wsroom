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
	store 		wsroom.Store
	upgrader = websocket.Upgrader{}
)

func init() {
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
}

func main() {
	var err error

	store, err = wsroom.NewMySqlStore("root:@tcp(localhost:3306)/wsroomtesting", "wsroom")
	if err != nil {
		log.Fatal(err)
	}

	room, err := store.New("notifications", 256)
	log.Println(room)
	switch err {
	case nil:

	case wsroom.ErrRoomAlreadyExists:
		log.Println("room already exists store")

	default:
		log.Fatal(err)
	}

	http.HandleFunc("/notifications/ws", connectWS)
	http.HandleFunc("/create_notifiaction", createNotification)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/", index)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	notRoom, _ := store.Get("notifications")
	tpl.ExecuteTemplate(w, "index.html", notRoom)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	log.Println("createNotication")
	msg := map[string]interface{} {"foo": "bar"}

	notRoom, err := store.Get("notifications")
	if err != nil {
		log.Panicln(err)
	}

	notRoom.CommunicationChannels.Broadcast <- msg
	log.Println("ended", notRoom, msg)
}

func connectWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	conn := wsroom.Connection {
		Key:	uuid.NewString(),
		Data: 	make(map[interface{}]interface{}),
		Conn: 	ws,
		Send: 	make(chan interface{}),
	}

	notRoom, _ := store.Get("notifications")

	err = notRoom.Subscribe(conn)
	if err != nil {
		log.Println(err)
		ws.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
	}
}