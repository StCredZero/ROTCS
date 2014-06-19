package main

import (
	"bytes"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type CstServer struct {
	// Registered connections.
	connections map[*connection]EntityID

	dropped map[EntityID](*connection)

	droppedQueue chan *connection

	load float64

	population int

	reconnectQueue chan reconnect

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
		reconnectQueue: make(chan reconnect, 1000),
		register:       make(chan *connection, 1000),
		dropped:        make(map[EntityID](*connection)),
		droppedQueue:   make(chan *connection, 1000),
		unregister:     make(chan *connection, 1000),
		connections:    make(map[*connection]EntityID),
	}
	srv.world = NewWorldGrid()

	for _, gc := range srv.world.spawnGrids {
		sg := srv.world.subgridAtGrid(gc)
		sg.lifeAllowed = false
	}

	srv.world.PutEntityAt(NewShipGuard(), Coord{10, 12})
	srv.world.PutEntityAt(NewShipGuard(), Coord{19, 9})
	srv.world.PutEntityAt(NewShipGuard(), Coord{19, 15})
	srv.world.PutEntityAt(NewShipGuard(), Coord{33, 9})
	srv.world.PutEntityAt(NewShipGuard(), Coord{33, 15})

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

	x := 1.0 - srv.load
	p := x * x

	if rand.Float64() < p {
		player := NewPlayer(c)
		entity, _ := srv.world.NewEntity(player)
		c.id = entity.EntityID()
		c.player = player
		srv.connections[c] = c.id
		buffer.WriteString(`{"type":"init",`)

		buffer.WriteString(`"uuid":"`)
		buffer.WriteString(c.id.String())
		buffer.WriteString(`",`)

		buffer.WriteString(`"approved":1}`)
		LogTrace("Initialized entity: ", entity)
	} else {
		buffer.WriteString(`{"type":"init",`)
		buffer.WriteString(`"pop":`)
		buffer.WriteString(strconv.FormatInt(int64(srv.population), 10))
		buffer.WriteRune(',')

		buffer.WriteString(`"load":`)
		buffer.WriteString(strconv.FormatFloat(srv.load, 'f', 2, 64))
		buffer.WriteString(`}`)
		LogTrace("refused registration")
	}
	c.send <- buffer.Bytes()

}

func (srv *CstServer) unregisterConnection(c *connection) {
	LogTrace("closing-final")
	srv.world.RemoveEntityID(c.id)
	delete(srv.connections, c)
}

func (srv *CstServer) reconnect(oldConn, newConn *connection) {
	LogTrace("reconnecting: ", oldConn.id)
	delete(srv.connections, oldConn)

	newConn.id = oldConn.id
	newConn.player = oldConn.player

	srv.connections[newConn] = oldConn.id
	LogTrace("Reconnected player: ", newConn.id)
}

const ticksPerSec = 8
const tickSecs = 1.0 / ticksPerSec

func (srv *CstServer) runLoop() {
	var load [ticksPerSec]float64
	for {
		startTime := time.Now()
		runtime.Gosched()

	dropped:
		for {
			select {
			case c := <-srv.droppedQueue:
				srv.dropped[c.id] = c
			default:
				break dropped
			}
		}

		now := time.Now()
		for id, c := range srv.dropped {
			if c.deadline.Add(time.Second * 20).After(now) {
				delete(srv.dropped, id)
			}
		}

	reconnect:
		for {
			select {
			case rc := <-srv.reconnectQueue:
				oldConn, present := srv.dropped[rc.oldId]
				if present {
					delete(srv.dropped, rc.oldId)
					srv.reconnect(oldConn, rc.newConn)
				}
			default:
				break reconnect
			}
		}
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
			//LogProfile("Players, Load: ", srv.population, srv.load)
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
	defer func() { srv.droppedQueue <- c }()
	go c.writer()
	c.reader(srv)
}
