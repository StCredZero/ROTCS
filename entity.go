package main

import (
	"bytes"
	"fmt"
	"math"
)

type EntityID uint32

type Creature interface {
	EntityID() EntityID
	Coord() Coord
	Detect(Creature)
	IsPlayer() bool
	Move(GridKeeper, GridProcessor)
	MoveCommit()
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	WriteFor(Creature, *bytes.Buffer)
}

type Entity struct {
	ID       EntityID
	Location Coord
	Symbol   rune
}

func (ntt *Entity) Coord() Coord {
	return ntt.Location
}

func (ntt *Entity) Detect(player Creature) {
}

func (ntt *Entity) SetCoord(coord Coord) {
	ntt.Location = coord
}

func (ntt *Entity) EntityID() EntityID {
	return ntt.ID
}

func (ntt *Entity) IsPlayer() bool {
	return false
}

func (ntt *Entity) Move(grid GridKeeper, gproc GridProcessor) {
}

func (ntt *Entity) MoveCommit() {
}

func (ntt *Entity) SendDisplay(grid GridKeeper, gproc GridProcessor) {
}

func (self *Entity) WriteFor(player Creature, buffer *bytes.Buffer) {
	self.Location.WriteDisplay(player, buffer)
	buffer.WriteString(`:{"symbol":"`)
	buffer.WriteRune(self.Symbol)
	buffer.WriteString(`"}`)
	self.Detect(player)
}

func EntityIDGenerator(lastId EntityID) chan (EntityID) {
	next := make(chan EntityID)
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
	return Creature(&Player{
		Entity:     entity,
		Moves:      "",
		Connection: c,
	})
}

func (ntt *Player) Coord() Coord {
	return ntt.Location
}

func (ntt *Player) IsPlayer() bool {
	return true
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
	if grid.OutOfBounds(newLoc) {
		grid.DeferMove(ntt)
		return
	}
	if grid.EmptyAt(newLoc) && gproc.WalkableAt(newLoc) {
		grid.MoveEntity(ntt, newLoc)
	}
}

func (ntt *Player) MoveCommit() {
	if len(ntt.Moves) > 0 {
		ntt.Moves = ntt.Moves[1:]
	}
}

func (ntt *Player) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	var buffer bytes.Buffer
	gproc.WriteDisplay(ntt, &buffer)
	ntt.Connection.send <- buffer.Bytes()
	ntt.LastUpdateLoc = ntt.Location
}

type detection struct {
	id   EntityID
	dist int64
}

type Monster struct {
	detections chan detection
	Entity
}

func NewMonster(c *connection) Creature {
	entity := Entity{
		ID:     c.id,
		Symbol: '@',
	}
	return &Monster{
		Entity:     entity,
		detections: make(chan detection, (subgrid_width * subgrid_height)),
	}
}

func (ntt *Monster) Detect(player Creature) {
	loc1, loc2 := ntt.Coord(), player.Coord()
	dist := manhattanDist(loc1, loc2)
	if dist <= 7 {
		det := detection{
			id:   player.EntityID(),
			dist: dist,
		}
		ntt.detections <- det
	}
}

func (ntt *Monster) Move(grid GridKeeper, gproc GridProcessor) {
	var min, det detection
	min = detection{
		dist: math.MaxInt32,
	}
	done, found := false, false
	for !done {
		select {
		case det = <-ntt.detections:
			if det.dist < min.dist {
				min = det
				found = true
			}
		default:
			done = true
		}
	}
	if found {

	}
}
