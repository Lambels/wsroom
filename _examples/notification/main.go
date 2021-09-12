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
	store = wsroom.NewRuntimeStore()
	upgrader = websocket.Upgrader{}
)

func init() {
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
}

func main() {

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

	room, err := store.Get("notifications")
	if err != nil {
		log.Println(err)
		return
	}

	room.CommunicationChannels.Broadcast <- msg
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

	room, err := store.Get("notifications")
	if err != nil {
		log.Println(err)
		return
	}

	err = room.Subscribe(conn)
	if err != nil {
		log.Println(err)
		ws.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
	}
}