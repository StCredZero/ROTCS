package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"strings"
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

	AddMessage(string)
	CalcMove(GridKeeper) Coord
	CanDamage(Creature) bool
	CanSwapWith(Creature) bool
	ChangeHealth(int)
	Collided() bool
	CollideWall()
	CollideWith(Creature)
	Coord() Coord
	Detect(Creature)
	DisplayString() string
	GetSubgrid() *SubGrid
	HasMove(GridProcessor) bool
	Health() int
	Inbox() []string
	Initialized() bool
	IsPlayer() bool
	IsTransient() bool
	LastDispCoord() Coord
	Outbox() []string
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	SetEntityID(EntityID)
	SetHealth(int)
	SetInitialized(bool)
	SetSubgrid(*SubGrid)
	TickZero(GridProcessor) bool
	WriteFor(Creature, *bytes.Buffer)
}

type Entity struct {
	ID EntityID

	direction    rune
	health       int
	Init         bool
	Location     Coord
	MoveSchedule uint8
	subgrid      *SubGrid
	Symbol       rune
	TickOffset   uint64
}

func (ntt *Entity) AddMessage(msg string) {}
func (ntt *Entity) CalcMove(grid GridKeeper) Coord {
	return ntt.Location
}
func (ntt *Entity) CanDamage(c Creature) bool {
	return false
}
func (ntt *Entity) CanSwapWith(c Creature) bool {
	return false
}
func (ntt *Entity) ChangeHealth(delta int) {
	ntt.health += delta
}
func (ntt *Entity) Collided() bool {
	return false
}
func (ntt *Entity) CollideWall()               {}
func (ntt *Entity) CollideWith(other Creature) {}
func (ntt *Entity) Coord() Coord {
	return ntt.Location
}
func (ntt *Entity) Detect(player Creature) {}
func (ntt *Entity) DisplayString() string {
	return fmt.Sprintf("%X%X%X%X", ntt.ID[0], ntt.ID[1], ntt.ID[2], ntt.ID[3])
}
func (ntt *Entity) EntityID() EntityID {
	return ntt.ID
}
func (ntt *Entity) GetSubgrid() *SubGrid {
	return ntt.subgrid
}
func (ntt *Entity) HasMove(gproc GridProcessor) bool {
	if ntt.health <= 0 {
		return false
	}
	phase := uint8((gproc.TickNumber() + ntt.TickOffset) % 8)
	return ((ntt.MoveSchedule >> phase) & 0x01) != 0x00
}
func (ntt *Entity) Health() int {
	return ntt.health
}
func (ntt *Entity) Inbox() []string {
	return nil
}
func (ntt *Entity) Initialized() bool {
	return ntt.Init
}
func (ntt *Entity) IsPlayer() bool    { return false }
func (ntt *Entity) IsTransient() bool { return true }
func (ntt *Entity) LastDispCoord() Coord {
	return ntt.Location
}
func (ntt *Entity) LocAhead() Coord {
	return ntt.Location.MovedBy(ntt.direction)
}
func (ntt *Entity) LocLeft() Coord {
	return ntt.Location.MovedBy(leftOf(ntt.direction))
}
func (ntt *Entity) LocRight() Coord {
	return ntt.Location.MovedBy(rightOf(ntt.direction))
}
func (ntt *Entity) Outbox() []string {
	return nil
}
func (ntt *Entity) SendDisplay(grid GridKeeper, gproc GridProcessor) {}
func (ntt *Entity) SetCoord(coord Coord) {
	ntt.Location = coord
}
func (ntt *Entity) SetEntityID(id EntityID) {
	ntt.ID = id
}
func (ntt *Entity) SetHealth(x int) {
	ntt.health = x
}
func (ntt *Entity) SetInitialized(flag bool) {
	ntt.Init = flag
}
func (ntt *Entity) SetSubgrid(grid *SubGrid) {
	ntt.subgrid = grid
}
func (ntt *Entity) TickZero(gproc GridProcessor) bool {
	phase := (gproc.TickNumber() + ntt.TickOffset) % 23
	return phase == 0
}
func (ntt *Entity) TurnLeft() {
	ntt.direction = leftOf(ntt.direction)
}
func (ntt *Entity) TurnRight() {
	ntt.direction = rightOf(ntt.direction)
}
func (self *Entity) WriteFor(player Creature, buffer *bytes.Buffer) {
	self.Location.WriteDisplay(player, buffer)
	buffer.WriteString(`:{"symbol":"`)
	buffer.WriteRune(self.Symbol)
	buffer.WriteString(`"}`)
}

type Player struct {
	Entity
	collided      bool
	Connection    *connection
	inbox         []string
	LastUpdateLoc Coord
	Moves         string
	outbox        []string
}

func NewPlayer(c *connection) *Player {
	entity := Entity{
		health:       80,
		ID:           c.id,
		Symbol:       '@',
		MoveSchedule: 0xFF,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return &Player{
		Connection: c,
		Entity:     entity,
		inbox:      make([]string, 0, (subgrid_width * subgrid_height)),
		Moves:      "",
		outbox:     make([]string, 0, 20),
	}
}

func (ntt *Player) AddMessage(msg string) {
	ntt.inbox = append(ntt.inbox, msg)
}
func (ntt *Player) CalcMove(grid GridKeeper) Coord {
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
	if len(ntt.Moves) > 0 {
		ntt.Moves = ntt.Moves[1:]
	}
	if move != '0' {
		ntt.direction = move
	}
	return loc.MovedBy(move)
}
func (ntt *Player) CanDamage(other Creature) bool {
	return !other.IsPlayer()
}
func (ntt *Player) CanSwapWith(other Creature) bool {
	return other.IsPlayer()
}
func (ntt *Player) Collided() bool {
	return ntt.collided
}
func (ntt *Player) CollideWall() {
	ntt.collided = true
}
func (ntt *Player) CollideWith(other Creature) {
	ntt.collided = true
	if ntt.CanDamage(other) {
		other.ChangeHealth(-1)
		ntt.AddMessage("hit monster")
	}
}
func (ntt *Player) Detect(player Creature) {
	//if player.IsPlayer() {
	for _, message := range player.Outbox() {
		ntt.Connection.send <- []byte(message)
	}
	//}
}
func (ntt *Player) FormattedMessage(msg string) string {
	s := []string{ntt.DisplayString(), `: `, msg}
	return strings.Join(s, "")
}
func (ntt *Player) Inbox() []string {
	return ntt.inbox
}
func (ntt *Player) IsPlayer() bool       { return true }
func (ntt *Player) IsTransient() bool    { return false }
func (ntt *Player) LastDispCoord() Coord { return ntt.LastUpdateLoc }
func (ntt *Player) Outbox() []string {
	return ntt.outbox
}
func (ntt *Player) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	LogTrace("Start SendDisplay ", ntt.Location)
	var buffer bytes.Buffer
	grid.WriteDisplay(ntt, gproc, &buffer)
	ntt.Connection.send <- buffer.Bytes()
	ntt.LastUpdateLoc = ntt.Location
	ntt.collided = false
	ntt.inbox = ntt.inbox[:0]
	ntt.outbox = ntt.outbox[:0]
	LogTrace("End SendDisplay ", ntt.Location)
}

type detection struct {
	id   EntityID
	loc  Coord
	dist float64
}

type Monster struct {
	Entity
	detections chan detection
	state      int
}

const mstStart int = 0
const mstToWall int = 1
const mstFollow int = 2

func NewMonster(id EntityID) Creature {
	entity := Entity{
		health:       10,
		ID:           id,
		Symbol:       '%',
		MoveSchedule: 0x55,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return &Monster{
		Entity:     entity,
		detections: make(chan detection, (subgrid_width * subgrid_height)),
		state:      mstStart,
	}
}

func (ntt *Monster) CalcMove(grid GridKeeper) Coord {
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
		ntt.state = mstStart
		openAt := func(coord Coord) bool {
			return grid.WalkableAt(coord)
		}
		path, pathFound := astarSearch(distance, openAt, neighbors4, ntt.Coord(), min.loc, 100)
		if pathFound {
			return path[0]
		}
	} else {
		stay := ntt.Location
		switch ntt.state {
		case mstStart:
			ntt.direction = int2dir(grid.RNG().Intn(4))
			ntt.state = mstToWall
		case mstToWall:
			ahead := ntt.LocAhead()
			if !grid.WalkableAt(ahead) {
				ntt.TurnLeft()
				ntt.state = mstFollow
				return stay
			}
			return ahead
		case mstFollow:
			ahead := ntt.LocAhead()
			right := ntt.LocRight()
			left := ntt.LocLeft()
			if grid.PassableAt(ahead) && !grid.PassableAt(right) {
				return ahead
			} else if grid.PassableAt(right) {
				ntt.TurnRight()
				return ntt.LocAhead()
			} else if !grid.PassableAt(right) && !grid.PassableAt(ahead) && !grid.PassableAt(left) {
				ntt.TurnRight()
				ntt.TurnRight()
				return stay
			} else if !grid.PassableAt(right) && !grid.PassableAt(ahead) && grid.PassableAt(left) {
				ntt.TurnLeft()
				return stay
			} else {
				ntt.TurnRight()
				return stay
			}
		}
	}
	return ntt.Location
}
func (ntt *Monster) CanDamage(other Creature) bool {
	return other.IsPlayer()
}
func (ntt *Monster) CollideWith(other Creature) {
	if ntt.CanDamage(other) {
		other.ChangeHealth(-1)
		other.AddMessage("hit by monster")
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
func (ntt *Monster) DisplayString() string {
	return "Exo"
}
