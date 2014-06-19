package main

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type moveRequest struct {
	direction rune
	timestamp uint64
}

func newConnection(ws *websocket.Conn) *connection {
	return &connection{
		isOpen: true,
		send:   make(chan []byte, 256),
		ws:     ws,
	}
}

type connection struct {
	id EntityID

	deadline time.Time

	IsBlurred bool

	isOpen bool

	outbox []string

	player *Player

	// Buffered channel of outbound messages.
	send chan []byte

	// The websocket connection.
	ws *websocket.Conn
}

type reconnect struct {
	oldId   EntityID
	newConn *connection
}

func (c *connection) reader(srv *CstServer) {

	defer func() {
		LogTrace("closing reader")
		c.isOpen = false
	}()

readerLoop:
	for c.isOpen {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break readerLoop
		}
		tokens := regexp.MustCompile(":").Split(string(message[:]), 3)
		timestamp, err := strconv.ParseUint(tokens[0], 10, 64)
		if err != nil {
			break readerLoop
		}
		cmdType, data := tokens[1], tokens[2]
		LogTrace("Got:", data, timestamp)
		if strings.EqualFold(cmdType, "mv") {
			for _, mv := range data {
				c.player.moveQueue <- moveRequest{mv, timestamp}
			}
		} else if strings.EqualFold(cmdType, "ch") {
			//c.player.outbox = append(c.player.outbox, data)
		} else if strings.EqualFold(cmdType, "bl") {
			flag, err := strconv.ParseUint(data, 10, 64)
			if err != nil {
				break readerLoop
			}
			c.IsBlurred = flag != 0
		} else if strings.EqualFold(cmdType, "li") {
			c.player.SetFlag(LifeCellTogl)
		} else if strings.EqualFold(cmdType, "al") {
			c.player.SetFlag(LifeActivateTogl)
		} else if strings.EqualFold(cmdType, "reconnect") {
			id, err := uuid.FromString(data)
			if err != nil {
				srv.reconnectQueue <- reconnect{EntityID(id), c}
			}
		}
		runtime.Gosched()
	}

}

func (c *connection) writer() {

	defer func() {
		LogTrace("closing writer")
		c.isOpen = false
		c.ws.Close()
	}()

writerLoop:
	for message := range c.send {
		if !c.isOpen {
			break writerLoop
		}
		LogTrace("writer about to WriteMessage", c.id)
		c.deadline = time.Now().Add(time.Duration(time.Millisecond * 1200))
		err1 := c.ws.SetWriteDeadline(c.deadline)
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
}
