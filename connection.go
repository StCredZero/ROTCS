package main

import "github.com/gorilla/websocket"
import "runtime"

func newConnection(ws *websocket.Conn) *connection {
	return &connection{
		closeConn: make(chan empty, 8),
		isOpen:    true,
		ws:        ws,
		send:      make(chan []byte, 256),
		moveQueue: make(chan string, 10),
	}
}

type connection struct {
	closeConn chan empty

	// The websocket connection.
	ws *websocket.Conn

	id EntityID

	isOpen bool

	moveQueue chan string

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader(srv *CstServer) {
readerLoop:
	for c.isOpen {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		TRACE.Println("Got:", s)
		if err != nil {
			c.isOpen = false
			break readerLoop
		}
		c.moveQueue <- s
	}
	TRACE.Println("closing reader")
	c.closeConn <- empty{}
}

func (c *connection) writer() {
writerLoop:
	for message := range c.send {
		if !c.isOpen {
			break writerLoop
		}
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			c.isOpen = false
			break writerLoop
		}
	}
	TRACE.Println("closing writer")
	c.closeConn <- empty{}
}

func (c *connection) closer(srv *CstServer) {
closeLoop:
	for {
		select {
		case <-c.closeConn:
			TRACE.Println("doing close")
			c.isOpen = false
			c.ws.Close()
			break closeLoop
		default:
			runtime.Gosched()
		}
	}
}
