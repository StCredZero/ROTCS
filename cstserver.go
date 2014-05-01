package main

import (
	"bytes"
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

	world *WorldGrid

	dunGenCache *DunGenCache
}

func NewCstServer() *CstServer {
	entropy := DunGenEntropy([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 55, 13, 14, 15, 16})
	dgproto := DunGen{
		xsize:      subgrid_width,
		ysize:      subgrid_height,
		targetObj:  20,
		chanceRoom: 50,
	}
	var srv = CstServer{
		register:    make(chan *connection, 1000),
		unregister:  make(chan *connection, 1000),
		connections: make(map[*connection]EntityId),
		entityIdGen: EntityIdGenerator(0),
		dunGenCache: NewDunGenCache(1000, entropy, dgproto),
	}

	srv.world = NewWorldGrid()
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

func updateFn(grid GridKeeper, ntt *Entity, gproc GridProcessor) {

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
	if grid.EmptyAt(newLoc) && gproc.WalkableAt(newLoc) {
		grid.MoveEntity(ntt, newLoc)
	}

	var buffer bytes.Buffer
	gproc.WriteDisplay(ntt, &buffer)
	ntt.Connection.send <- buffer.Bytes()
}

func (self *CstServer) DungeonAt(coord Coord) int {
	return self.dunGenCache.DungeonAt(coord)
}

func (srv *CstServer) ProcessEntities(gridUpdate GridUpdateFn, sref *CstServer) {
	srv.world.UpdateEntities(gridUpdate, srv)
}

func (srv *CstServer) WalkableAt(coord Coord) bool {
	return srv.dunGenCache.WalkableAt(coord)
}

func (srv *CstServer) WriteDisplay(ntt *Entity, buffer *bytes.Buffer) {
	buffer.WriteString(`{"type":"update","data":{`)
	buffer.WriteString(`"maptype":"basic",`)
	buffer.WriteString(`"map":`)
	srv.dunGenCache.WriteBasicMap(ntt, buffer)
	buffer.WriteRune(',')
	buffer.WriteString(`"entities":{`)
	srv.world.WriteEntities(ntt, buffer)
	buffer.WriteString(`}}}`)

}

func (srv *CstServer) registerConnection(c *connection) {
	if debugFlag {
		println("starting register")
	}
	srv.connections[c] = c.id

	newPlayer := NewPlayer(c)
	newPlayer, _ = srv.world.NewEntity(newPlayer)
	if debugFlag {
		fmt.Println("Initialized entity: ", *newPlayer)
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
