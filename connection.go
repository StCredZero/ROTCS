package main

import (
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

func newConnection(ws *websocket.Conn) *connection {
	return &connection{
		isOpen:    true,
		ws:        ws,
		send:      make(chan []byte, 256),
		moveQueue: make(chan string, 10),
	}
}

type connection struct {
	id EntityID

	isOpen bool

	moveQueue chan string

	// Buffered channel of outbound messages.
	send chan []byte

	// The websocket connection.
	ws *websocket.Conn
}

func (c *connection) reader(srv *CstServer) {
readerLoop:
	for c.isOpen {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		LogTrace("Got:", s)
		if err != nil {
			break readerLoop
		}
		c.moveQueue <- s
		runtime.Gosched()
	}
	LogTrace("closing reader")
	c.isOpen = false
}

func (c *connection) writer() {
writerLoop:
	for message := range c.send {
		if !c.isOpen {
			break writerLoop
		}
		LogTrace("writer about to WriteMessage", c.id)
		deadline := time.Now().Add(time.Duration(time.Millisecond * 120))
		err1 := c.ws.SetWriteDeadline(deadline)
		if err1 != nil {
			break writerLoop
		}
		err2 := c.ws.WriteMessage(websocket.TextMessage, message)
		LogTrace("wrote WriteMessage", c.id)
		if err2 != nil {
			break writerLoop
		}
		runtime.Gosched()
	}
	LogTrace("closing writer")
	c.isOpen = false
	c.ws.Close()
}

/*func (c *connection) closer(srv *CstServer) {
	select {
	case <-c.closeConn:
		LogTrace("doing close")
		c.isOpen = false
		c.ws.Close()
	}
}*/
