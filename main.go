package main

import (
	//"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"go/build"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan []byte

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			delete(h.connections, c)
			close(c.send)
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
				}
			}
		}
	}
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader(h *hub) {
	for {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		fmt.Printf("Got message: %q %s", message, s)
		if err != nil {
			break
		}
		h.broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func defaultAssetPath() string {
	p, err := build.Default.Import("github.com/StCredZero/casterly", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

func homeHandler(c http.ResponseWriter, req *http.Request, homeTempl *template.Template) {
	homeTempl.Execute(c, req.Host)
}

func (h *hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader(h)
}

func main() {
	flag.Parse()

	var h = hub{
		broadcast:   make(chan []byte),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
	}
	go h.run()

	var addr = flag.String("addr", ":8080", "http service address")
	var assets = flag.String("assets", defaultAssetPath(), "path to assets")
	var homeTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "index.html")))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(w, r, homeTempl)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		h.wsHandler(w, r)
	})
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
