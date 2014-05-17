package main

import (
	"bytes"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

type CstServer struct {
	// Registered connections.
	connections map[*connection]EntityID

	load float64

	population int

	// Register requests from the connections.
	register chan *connection

	tickNumber uint64

	// Unregister requests from connections.
	unregister chan *connection

	//entityIdGen chan EntityID

	world *WorldGrid
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

func (srv *CstServer) ServerLoad() float64 {
	return srv.load
}

func (srv *CstServer) ServerPopulation() int {
	return srv.population
}

func (srv *CstServer) TickNumber() uint64 {
	return srv.tickNumber
}

func (srv *CstServer) registerConnection(c *connection) {
	LogTrace("starting register")
	var buffer bytes.Buffer
	if srv.load > 0.95 {
		buffer.WriteString(`{"type":"init"}`)
		LogTrace("refused registration")
	} else {
		player := NewPlayer(c)
		entity, _ := srv.world.NewEntity(player)
		c.id = entity.EntityID()
		c.player = player
		srv.connections[c] = c.id
		buffer.WriteString(`{"type":"init","approved":1}`)
		LogTrace("Initialized entity: ", entity)
	}
	c.send <- buffer.Bytes()

}

func (srv *CstServer) unregisterConnection(c *connection) {
	LogTrace("closing-final")
	srv.world.RemoveEntityID(c.id)
	delete(srv.connections, c)
}

const ticksPerSec = 8
const tickSecs = 1.0 / ticksPerSec

func (srv *CstServer) runLoop() {
	var load [ticksPerSec]float64
	for {
		startTime := time.Now()
		runtime.Gosched()

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

		runtime.GC()

		tickDuration := time.Since(startTime).Seconds()
		phase := int(srv.tickNumber % ticksPerSec)
		load[phase] = tickDuration / tickSecs

		if phase == 0 {
			var sum float64
			for i := 0; i < ticksPerSec; i++ {
				sum += load[i]
			}
			srv.load = sum / ticksPerSec
			srv.population = srv.world.playerCount()
			//message := fmt.Sprintf("Load: %f", avg)
			LogProfile("Players, Load: ", srv.population, srv.load)
		}

		if tickDuration < tickSecs {
			time.Sleep(time.Duration((tickSecs-tickDuration)*1000) * time.Millisecond)
		}
		srv.tickNumber++
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
