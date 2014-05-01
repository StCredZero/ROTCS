package main

import (
	"bytes"
	"fmt"
)

type EntityId uint32

type Mover interface {
	Move(GridKeeper, GridProcessor)
	PopMoveQueue()
}

type Displayer interface {
	ID() EntityId
	Coord() Coord
	SendDisplay(GridKeeper, GridProcessor)
}

type Entity struct {
	Id            EntityId
	Connection    *connection
	Location      Coord
	LastUpdateLoc Coord
	Symbol        rune
	Moves         string
}

func NewPlayer(c *connection) *Entity {
	return &Entity{
		Id:         c.id,
		Moves:      "",
		Connection: c,
		Symbol:     '@',
	}
}

func (ntt *Entity) Coord() Coord {
	return ntt.Location
}

func (ntt *Entity) ID() EntityId {
	return ntt.Id
}

func (ntt *Entity) Move(grid GridKeeper, gproc GridProcessor) {

	select {
	case moves := <-ntt.Connection.moveQueue:
		ntt.Moves = moves
	default:
	}

	var move rune = '0'
	for _, move = range ntt.Moves {
		break
	}

	newLoc := updateLoc(move, ntt.Location)
	if debugFlag {
		fmt.Println(newLoc)
	}
	if grid.EmptyAt(newLoc) && gproc.WalkableAt(newLoc) {
		grid.MoveEntity(ntt, newLoc)
	}
}

func (ntt *Entity) PopMoveQueue() {
	if len(ntt.Moves) > 0 {
		ntt.Moves = ntt.Moves[1:]
	}
}

func (ntt *Entity) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	var buffer bytes.Buffer
	gproc.WriteDisplay(ntt, &buffer)
	ntt.Connection.send <- buffer.Bytes()
}

func (self *Entity) WriteEntities(player Displayer, buffer *bytes.Buffer) {
	self.Location.WriteDisplay(player, buffer)
	buffer.WriteString(`:{"symbol":"`)
	buffer.WriteRune(self.Symbol)
	buffer.WriteString(`"}`)
}

func EntityIdGenerator(lastId EntityId) chan (EntityId) {
	next := make(chan EntityId)
	id := lastId + 1
	go func() {
		for {
			next <- id
			id++
		}
	}()
	return next
}
