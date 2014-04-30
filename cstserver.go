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
	connections map[*connection]EntityId

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	entityIdGen chan EntityId

	world SubGrid

	dunGenCache *DunGenCache
}

func NewCstServer() *CstServer {
	var srv = CstServer{
		register:    make(chan *connection, 1000),
		unregister:  make(chan *connection, 1000),
		connections: make(map[*connection]EntityId),
		entityIdGen: EntityIdGenerator(0),
	}

	srv.world = SubGrid{
		GridCoord:   GridCoord{0, 0},
		Grid:        make(map[Coord]EntityId),
		Entities:    make(map[EntityId]Entity),
		ParentQueue: make(chan EntityId, (subgrid_width * subgrid_height)),
	}
	return &srv
}

func updateLoc(move rune, loc Coord) Coord {
	if move == 'n' {
		return Coord{loc.x, loc.y - 1}
	} else if move == 's' {
		return Coord{loc.x, loc.y + 1}
	} else if move == 'w' {
		return Coord{loc.x - 1, loc.y}
	} else if move == 'e' {
		return Coord{loc.x + 1, loc.y}
	}
	return loc
}

func updateFn(grid GridKeeper, ntt Entity, gproc GridProcessor) {

	select {
	case moves := <-ntt.Connection.moveQueue:
		ntt.Moves = moves
	default:
	}

	var move rune = '0'
	for _, move = range ntt.Moves {
		ntt.Moves = ntt.Moves[1:]
		break
	}

	newLoc := updateLoc(move, ntt.Location)
	if debugFlag {
		fmt.Println(newLoc)
	}
	grid.MoveEntity(ntt, newLoc)

	ntt.Connection.send <- []byte(grid.DisplayString())
}

func (srv *CstServer) ProcessEntities(gridUpdate GridUpdateFn, sref *CstServer) {
	srv.world.UpdateEntities(gridUpdate, srv)
}

func (srv *CstServer) registerConnection(c *connection) {
	if debugFlag {
		println("starting register")
	}
	srv.connections[c] = c.id

	newEntity := Entity{
		Id:         c.id,
		Moves:      "",
		Connection: c,
	}
	newEntity, _ = srv.world.NewEntity(newEntity)
	if debugFlag {
		fmt.Println("Initialized entity: ", newEntity)
	}
}

func (srv *CstServer) unregisterConnection(c *connection) {
	if debugFlag {
		println("closing-final")
	}
	srv.world.RemoveEntityId(c.id)
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
		srv.ProcessEntities(updateFn, srv)

		tickDuration := time.Since(startTime).Seconds()
		if tickDuration < 0.125 {
			load := tickDuration / 0.125
			fmt.Println("load: ", load)
			time.Sleep(time.Duration((0.125-tickDuration)*1000) * time.Millisecond)
		}
		runtime.Gosched()
	}
}

func (srv *CstServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	var id = <-srv.entityIdGen
	c := newConnection(ws, id)
	srv.register <- c
	defer func() { srv.unregister <- c }()
	go c.writer()
	c.reader(srv)
}
