package main

import "bytes"

type EntityId uint32

type Entity struct {
	Id         EntityId
	Location   Coord
	Symbol     rune
	Moves      string
	Connection *connection
}

func NewPlayer(c *connection) *Entity {
	return &Entity{
		Id:         c.id,
		Moves:      "",
		Connection: c,
		Symbol:     '@',
	}
}

func (self *Entity) WriteEntities(player *Entity, buffer *bytes.Buffer) {
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
