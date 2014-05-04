package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"runtime"
	"time"
)

type CstServer struct {
	// Registered connections.
	connections map[*connection]EntityID

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	//entityIdGen chan EntityID

	world *WorldGrid

	//dunGenCache *DunGenCache

	tickNumber uint64
}

func NewCstServer() *CstServer {

	var srv = CstServer{
		register:    make(chan *connection, 1000),
		unregister:  make(chan *connection, 1000),
		connections: make(map[*connection]EntityID),
	}
	srv.world = NewWorldGrid()
	return &srv
}

func (srv *CstServer) TickNumber() uint64 {
	return srv.tickNumber
}

func (srv *CstServer) registerConnection(c *connection) {
	if debugFlag {
		println("starting register")
	}
	player := NewPlayer(c)
	entity, _ := srv.world.NewEntity(player)
	c.id = entity.EntityID()
	srv.connections[c] = c.id
	if debugFlag {
		fmt.Println("Initialized entity: ", entity)
	}
}

func (srv *CstServer) unregisterConnection(c *connection) {
	if debugFlag {
		println("closing-final")
	}
	srv.world.RemoveEntityID(c.id)
	delete(srv.connections, c)
	close(c.send)
}

func (srv *CstServer) runLoop() {
	for {
		startTime := time.Now()

	register:
		for {
			select {
			case c := <-srv.register:
				srv.registerConnection(c)
			default:
				break register
			}
		}
	unregister:
		for {
			select {
			case c := <-srv.unregister:
				srv.unregisterConnection(c)
			default:
				break unregister
			}
		}
		prepop, cull := srv.world.prepopCullGrids()

		srv.world.prepopulateGrids(prepop)
		srv.world.cullGrids(cull)
		srv.world.UpdateMovers(srv)
		srv.world.SendDisplays(srv)
		srv.world.discardEmpty()

		tickDuration := time.Since(startTime).Seconds()
		if tickDuration < 0.125 {
			load := tickDuration / 0.125
			fmt.Println("load: ", load)
			time.Sleep(time.Duration((0.125-tickDuration)*1000) * time.Millisecond)
		}
		srv.tickNumber++
		runtime.Gosched()
	}
}

func (srv *CstServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	c := newConnection(ws)
	srv.register <- c
	defer func() { srv.unregister <- c }()
	go c.writer()
	c.reader(srv)
}
