package main

import "github.com/gorilla/websocket"

func newConnection(ws *websocket.Conn) *connection {
	return &connection{
		send:      make(chan []byte, 256),
		moveQueue: make(chan string, 10),
		ws:        ws,
	}
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	moveQueue chan string

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader(srv *CstServer) {
	for {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		print("got: ")
		println(s)
		if err != nil {
			break
		}
		c.moveQueue <- s
	}
	println("closing reader")
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	println("closing writer")
	c.ws.Close()
}
