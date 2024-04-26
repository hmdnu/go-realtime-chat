package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/hmdnubaidillah/go-routine-practice/types"
	gubrak "github.com/novalagung/gubrak/v2"
)

const (
	MESSAGE_NEW_USER = "New User"
	MESSAGE_CHAT     = "Chat"
	MESSAGE_LEAVE    = "Leave"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var connections = make([]*types.WebSocketConnection, 0)

func handleIO(currentConn types.WebSocketConnection) {

	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadCastMessage(&currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := types.SocketPayload{}

		err := currentConn.ReadJSON(&payload)

		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadCastMessage(&currentConn, MESSAGE_LEAVE, "")
				ejectConnection(&currentConn)
				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadCastMessage(&currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func broadCastMessage(currentConn *types.WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		eachConn.WriteJSON(types.SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}

func ejectConnection(currentConn *types.WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *types.WebSocketConnection) bool {
		return each == currentConn
	}).Result()

	connections = filtered.([]*types.WebSocketConnection)
}

func middleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Origin", "*")
		handler(w, req)
	}
}

func main() {
	http.HandleFunc("/", middleware(func(w http.ResponseWriter, req *http.Request) {
		content, err := os.ReadFile("./client/index.html")

		if err != nil {
			http.Error(w, "could not open requested file", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%s", content)
	}))

	http.HandleFunc("/ws", middleware(func(w http.ResponseWriter, req *http.Request) {
		upgrader.CheckOrigin = func(req *http.Request) bool {
			return true
		}

		// websocket
		currentGorillaConn, err := upgrader.Upgrade(w, req, w.Header())

		if err != nil {
			// http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			log.Println("error mas", err)
			return
		}

		username := req.URL.Query().Get("username")

		currentConn := types.WebSocketConnection{Conn: currentGorillaConn, Username: username}
		connections = append(connections, &currentConn)

		go handleIO(currentConn)
	}))

	fmt.Print("server starting at :8080\n")
	http.ListenAndServe(":8080", nil)
}
