package main

import "github.com/gorilla/websocket"

func newConnection(ws *websocket.Conn) *connection {
	return &connection{
		ws:        ws,
		send:      make(chan []byte, 256),
		moveQueue: make(chan string, 10),
	}
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	id EntityID

	moveQueue chan string

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader(srv *CstServer) {
	for {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		TRACE.Println("Got:", s)
		if err != nil {
			break
		}
		c.moveQueue <- s
	}
	TRACE.Println("closing reader")
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	TRACE.Println("closing writer")
	c.ws.Close()
}
