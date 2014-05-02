package main

import (
	"bytes"
	"fmt"
)

type EntityId uint32

type Creature interface {
	EntityID() EntityId
	Coord() Coord
	Move(GridKeeper, GridProcessor)
	PopMoveQueue()
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	WriteFor(Creature, *bytes.Buffer)
}

type Entity struct {
	ID       EntityId
	Location Coord
	Symbol   rune
}

func (ntt *Entity) Coord() Coord {
	return ntt.Location
}

func (ntt *Entity) SetCoord(coord Coord) {
	ntt.Location = coord
}

func (ntt *Entity) EntityID() EntityId {
	return ntt.ID
}

func (ntt *Entity) Move(grid GridKeeper, gproc GridProcessor) {
}

func (ntt *Entity) PopMoveQueue() {
}

func (ntt *Entity) SendDisplay(grid GridKeeper, gproc GridProcessor) {
}

func (self *Entity) WriteFor(player Creature, buffer *bytes.Buffer) {
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

type Player struct {
	Connection    *connection
	LastUpdateLoc Coord
	Moves         string
	Entity
}

func NewPlayer(c *connection) Creature {
	entity := Entity{
		ID:     c.id,
		Symbol: '@',
	}
	return &Player{
		Entity:     entity,
		Moves:      "",
		Connection: c,
	}
}

func (ntt *Player) Move(grid GridKeeper, gproc GridProcessor) {

	select {
	case moves := <-ntt.Connection.moveQueue:
		ntt.Moves = moves
	default:
	}
	loc := ntt.Location
	move := '0'
	for _, move = range ntt.Moves {
		break
	}

	newLoc := loc.MovedBy(move)
	if debugFlag {
		fmt.Println(newLoc)
	}
	if grid.EmptyAt(newLoc) && gproc.WalkableAt(newLoc) {
		grid.MoveEntity(ntt, newLoc)
	}
}

func (ntt *Player) PopMoveQueue() {
	if len(ntt.Moves) > 0 {
		ntt.Moves = ntt.Moves[1:]
	}
}

func (ntt *Player) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	var buffer bytes.Buffer
	gproc.WriteDisplay(ntt, &buffer)
	ntt.Connection.send <- buffer.Bytes()
}
