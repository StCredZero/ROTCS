package main

import (
	"runtime"
	"strings"
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

	player Creature

	// Buffered channel of outbound messages.
	send chan []byte

	// The websocket connection.
	ws *websocket.Conn
}

func (c *connection) reader(srv *CstServer) {
readerLoop:
	for c.isOpen {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break readerLoop
		}
		msgtype := string(message[0:2])
		s := string(message[3:])
		LogTrace("Got:", s)
		if strings.EqualFold(msgtype, "mv") {
			c.moveQueue <- s
		} else if strings.EqualFold(msgtype, "ch") {
			subgrid := c.player.GetSubgrid()
			subgrid.chatQueue <- s
			println(s)
		}
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
