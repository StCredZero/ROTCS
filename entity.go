package main

import (
	"bytes"
	"math"
	"math/rand"
	"time"

	"github.com/satori/go.uuid"
)

var offsetRNG = rand.New(rand.NewSource(time.Now().UnixNano()))

type EntityID [16]byte

func NewEntityID() EntityID {
	return EntityID(uuid.NewV1())
}

type Creature interface {
	EntityID() EntityID
	Coord() Coord
	Detect(Creature)
	HasMove(GridProcessor) bool
	Initialized() bool
	SetInitialized(bool)
	IsPlayer() bool
	IsTransient() bool
	LastDispCoord() Coord
	Move(GridKeeper, GridProcessor)
	MoveCommit()
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	SetEntityID(EntityID)
	TickZero(GridProcessor) bool
	WriteFor(Creature, *bytes.Buffer)
}

type Entity struct {
	ID           EntityID
	Init         bool
	Location     Coord
	Symbol       rune
	MoveSchedule uint8
	TickOffset   uint64
}

func (ntt *Entity) Coord() Coord {
	return ntt.Location
}
func (ntt *Entity) Detect(player Creature) {}
func (ntt *Entity) Initialized() bool {
	return ntt.Init
}
func (ntt *Entity) SetInitialized(flag bool) {
	ntt.Init = flag
}
func (ntt *Entity) SetCoord(coord Coord) {
	ntt.Location = coord
}
func (ntt *Entity) EntityID() EntityID {
	return ntt.ID
}
func (ntt *Entity) SetEntityID(id EntityID) {
	ntt.ID = id
}
func (ntt *Entity) HasMove(gproc GridProcessor) bool {
	phase := uint8((gproc.TickNumber() + ntt.TickOffset) % 8)
	return ((ntt.MoveSchedule >> phase) & 0x01) != 0x00
}
func (ntt *Entity) IsPlayer() bool    { return false }
func (ntt *Entity) IsTransient() bool { return true }
func (ntt *Entity) LastDispCoord() Coord {
	return ntt.Location
}
func (ntt *Entity) Move(grid GridKeeper, gproc GridProcessor)        {}
func (ntt *Entity) MoveCommit()                                      {}
func (ntt *Entity) SendDisplay(grid GridKeeper, gproc GridProcessor) {}
func (ntt *Entity) TickZero(gproc GridProcessor) bool {
	phase := (gproc.TickNumber() + ntt.TickOffset) % 23
	return phase == 0
}

func (self *Entity) WriteFor(player Creature, buffer *bytes.Buffer) {
	self.Location.WriteDisplay(player, buffer)
	buffer.WriteString(`:{"symbol":"`)
	buffer.WriteRune(self.Symbol)
	buffer.WriteString(`"}`)
}

type Player struct {
	Entity
	Connection    *connection
	LastUpdateLoc Coord
	Moves         string
}

func NewPlayer(c *connection) Creature {
	entity := Entity{
		ID:           c.id,
		Symbol:       '@',
		MoveSchedule: 0xFF,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return Creature(&Player{
		Entity:     entity,
		Moves:      "",
		Connection: c,
	})
}

func (ntt *Player) IsPlayer() bool       { return true }
func (ntt *Player) IsTransient() bool    { return false }
func (ntt *Player) LastDispCoord() Coord { return ntt.LastUpdateLoc }
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
	if grid.OutOfBounds(newLoc) {
		grid.DeferMove(ntt)
		return
	}
	if grid.EmptyAt(newLoc) && grid.WalkableAt(newLoc) {
		grid.MoveEntity(ntt, newLoc)
	} else {
		ntt.MoveCommit() //Don't move, but lose the []Moves item anyways
	}
}
func (ntt *Player) MoveCommit() {
	if len(ntt.Moves) > 0 {
		ntt.Moves = ntt.Moves[1:]
	}
}
func (ntt *Player) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	TRACE.Print("Start SendDisplay ")
	TRACE.Println(ntt.Location)
	var buffer bytes.Buffer
	grid.WriteDisplay(ntt, &buffer)
	ntt.Connection.send <- buffer.Bytes()
	ntt.LastUpdateLoc = ntt.Location
	TRACE.Println("End SendDisplay")
}

type detection struct {
	id   EntityID
	loc  Coord
	dist float64
}

type Monster struct {
	Entity
	detections chan detection
}

func NewMonster(id EntityID) Creature {
	entity := Entity{
		ID:           id,
		Symbol:       '%',
		MoveSchedule: 0x55,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return &Monster{
		Entity:     entity,
		detections: make(chan detection, (subgrid_width * subgrid_height)),
	}
}

func (ntt *Monster) Detect(player Creature) {
	loc1, loc2 := ntt.Coord(), player.Coord()
	dist := distance(loc1, loc2)
	if dist <= 7 {
		det := detection{
			id:   player.EntityID(),
			loc:  loc2,
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
	done, minFound := false, false
	for !done {
		select {
		case det = <-ntt.detections:
			if det.dist < min.dist {
				min = det
				minFound = true
			}
		default:
			done = true
		}
	}
	if minFound {
		openAt := func(coord Coord) bool {
			return grid.WalkableAt(coord)
		}
		path, pathFound := astarSearch(distance, openAt, neighbors4, ntt.Coord(), min.loc, 100)
		if pathFound {
			newLoc := path[0]
			if grid.OutOfBounds(newLoc) {
				grid.DeferMove(ntt)
				ntt.detections <- min
				return
			}
			if grid.EmptyAt(newLoc) && grid.WalkableAt(newLoc) {
				grid.MoveEntity(ntt, newLoc)
			}
		}
	}
}
