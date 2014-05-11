package main

import (
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

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
	defer func() {
		LogTrace("closing reader")
		c.closeConn <- empty{}
	}()
readerLoop:
	for c.isOpen {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		LogTrace("Got:", s)
		if err != nil {
			c.isOpen = false
			break readerLoop
		}
		c.moveQueue <- s
		runtime.Gosched()
	}
}

func (c *connection) writer() {
	defer func() {
		LogTrace("closing writer")
		c.closeConn <- empty{}
	}()
writerLoop:
	for message := range c.send {
		if !c.isOpen {
			break writerLoop
		}
		LogTrace("writer about to WriteMessage", c.id)
		deadline := time.Now().Add(time.Duration(time.Millisecond * 120))
		err1 := c.ws.SetWriteDeadline(deadline)
		if err1 != nil {
			c.isOpen = false
			break writerLoop
		}
		err2 := c.ws.WriteMessage(websocket.TextMessage, message)
		LogTrace("wrote WriteMessage", c.id)
		if err2 != nil {
			c.isOpen = false
			break writerLoop
		}
		runtime.Gosched()
	}
}

func (c *connection) closer(srv *CstServer) {
	select {
	case <-c.closeConn:
		LogTrace("doing close")
		c.isOpen = false
		c.ws.Close()
	}
}
