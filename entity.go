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

type Entity interface {
	EntityID() EntityID

	AddMessage(string)
	CalcMove(GridKeeper) Coord
	CanDamage(Entity) bool
	CanSwapWith(Entity) bool
	ChangeHealth(int)
	Collided() bool
	CollideWall()
	CollisionFrom(other Entity)
	Coord() Coord
	DeathSpawn() (Entity, bool)
	Detect(Entity)
	DisplayString() string
	FormattedMessage(string) string
	GetSubgrid() *SubGrid
	HasMove(GridProcessor) bool
	Health() int
	Inbox() []string
	Initialized() bool
	IsDead() bool
	IsPlayer() bool
	IsTransient() bool
	IsWalkable() bool
	LastDispCoord() Coord
	Outbox() []string
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	SetCollided()
	SetEntityID(EntityID)
	SetHealth(int)
	SetInitialized(bool)
	SetSubgrid(*SubGrid)
	TickZero(GridProcessor) bool
	WriteFor(Entity, *bytes.Buffer)
}

type EntityT struct {
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

func (ntt *EntityT) AddMessage(msg string) {}
func (ntt *EntityT) CalcMove(grid GridKeeper) Coord {
	return ntt.Location
}
func (ntt *EntityT) CanDamage(c Entity) bool {
	return false
}
func (ntt *EntityT) CanSwapWith(c Entity) bool {
	return false
}
func (ntt *EntityT) ChangeHealth(delta int) {
	ntt.health += delta
}
func (ntt *EntityT) Collided() bool {
	return false
}
func (ntt *EntityT) CollideWall()         {}
func (ntt *EntityT) CollisionFrom(Entity) {}
func (ntt *EntityT) Coord() Coord {
	return ntt.Location
}
func (ntt *EntityT) DeathSpawn() (Entity, bool) {
	return nil, false
}
func (ntt *EntityT) Detect(player Entity) {}
func (ntt *EntityT) DisplayString() string {
	return fmt.Sprintf("%X%X%X%X", ntt.ID[0], ntt.ID[1], ntt.ID[2], ntt.ID[3])
}
func (ntt *EntityT) EntityID() EntityID {
	return ntt.ID
}
func (ntt *EntityT) FormattedMessage(s string) string {
	return s
}
func (ntt *EntityT) GetSubgrid() *SubGrid {
	return ntt.subgrid
}
func (ntt *EntityT) HasMove(gproc GridProcessor) bool {
	if ntt.health <= 0 {
		return false
	}
	phase := uint8((gproc.TickNumber() + ntt.TickOffset) % 8)
	return ((ntt.MoveSchedule >> phase) & 0x01) != 0x00
}
func (ntt *EntityT) Health() int {
	return ntt.health
}
func (ntt *EntityT) Inbox() []string {
	return nil
}
func (ntt *EntityT) Initialized() bool {
	return ntt.Init
}
func (ntt *EntityT) IsDead() bool      { return true }
func (ntt *EntityT) IsPlayer() bool    { return false }
func (ntt *EntityT) IsTransient() bool { return true }
func (ntt *EntityT) IsWalkable() bool  { return false }
func (ntt *EntityT) LastDispCoord() Coord {
	return ntt.Location
}
func (ntt *EntityT) LocAhead() Coord {
	return ntt.Location.MovedBy(ntt.direction)
}
func (ntt *EntityT) LocLeft() Coord {
	return ntt.Location.MovedBy(leftOf(ntt.direction))
}
func (ntt *EntityT) LocRight() Coord {
	return ntt.Location.MovedBy(rightOf(ntt.direction))
}
func (ntt *EntityT) LocRightRear() Coord {
	right := rightOf(ntt.direction)
	return ntt.Location.MovedBy(right).MovedBy(rightOf(right))
}
func (ntt *EntityT) Outbox() []string {
	return nil
}
func (ntt *EntityT) SendDisplay(grid GridKeeper, gproc GridProcessor) {}
func (ntt *EntityT) SetCollided()                                     {}
func (ntt *EntityT) SetCoord(coord Coord) {
	ntt.Location = coord
}
func (ntt *EntityT) SetEntityID(id EntityID) {
	ntt.ID = id
}
func (ntt *EntityT) SetHealth(x int) {
	ntt.health = x
}
func (ntt *EntityT) SetInitialized(flag bool) {
	ntt.Init = flag
}
func (ntt *EntityT) SetSubgrid(grid *SubGrid) {
	ntt.subgrid = grid
}
func (ntt *EntityT) TickZero(gproc GridProcessor) bool {
	phase := (gproc.TickNumber() + ntt.TickOffset) % 23
	return phase == 0
}
func (ntt *EntityT) TurnLeft() {
	ntt.direction = leftOf(ntt.direction)
}
func (ntt *EntityT) TurnRight() {
	ntt.direction = rightOf(ntt.direction)
}
func (self *EntityT) WriteFor(player Entity, buffer *bytes.Buffer) {
	self.Location.WriteDisplay(player, buffer)
	buffer.WriteString(`:{"symbol":"`)
	buffer.WriteRune(self.Symbol)
	buffer.WriteString(`"}`)
}

type Player struct {
	EntityT
	collided      bool
	Connection    *connection
	inbox         []string
	LastUpdateLoc Coord
	Moves         string
	outbox        []string
}

func NewPlayer(c *connection) *Player {
	entity := EntityT{
		health:       80,
		ID:           c.id,
		Symbol:       '@',
		MoveSchedule: 0xFF,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return &Player{
		Connection: c,
		EntityT:    entity,
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
func (ntt *Player) CanDamage(other Entity) bool {
	return !other.IsPlayer()
}
func (ntt *Player) CanSwapWith(other Entity) bool {
	return other.IsPlayer()
}
func (ntt *Player) Collided() bool {
	return ntt.collided
}
func (ntt *Player) CollideWall() {
	ntt.collided = true
}
func (ntt *Player) CollisionFrom(other Entity) {
	other.SetCollided()
	if other.CanDamage(ntt) {
		if other.Coord() == ntt.LocAhead() {
			ntt.ChangeHealth(-1)
			ntt.AddMessage("shield hit, damage -1")
		} else {
			ntt.ChangeHealth(-2)
			ntt.AddMessage("flank hit, damage -2")
		}
	}
}
func (ntt *Player) Detect(player Entity) {
	//if player.IsPlayer() {
	var buffer bytes.Buffer
	for _, message := range player.Outbox() {
		escaped := strings.Replace(message, `"`, `&quot;`, -1)
		buffer.WriteString(`{"type":"message","data":"`)
		buffer.WriteString(player.FormattedMessage(escaped))
		buffer.WriteString(`"}`)
		ntt.Connection.send <- []byte(buffer.Bytes())
		buffer.Truncate(0)
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
func (ntt *Player) IsDead() bool {
	return ntt.Health() <= -10
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
	EntityT
	detections chan detection
	state      int
}

const mstStart int = 0
const mstToWall int = 1
const mstFollow int = 2

func NewMonster(id EntityID) Entity {
	entity := EntityT{
		health:       8,
		ID:           id,
		Symbol:       '%',
		MoveSchedule: 0x55,
		TickOffset:   uint64(offsetRNG.Intn(23)),
	}
	return &Monster{
		EntityT:    entity,
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
			} else if !grid.PassableAt(ahead) {
				ntt.TurnLeft()
				return stay
			}
			return ahead
		case mstFollow:
			ahead := ntt.LocAhead()
			right := ntt.LocRight()
			left := ntt.LocLeft()
			rightRear := ntt.LocRightRear()
			if grid.WalkableAt(right) && grid.WalkableAt(ahead) &&
				grid.WalkableAt(left) && grid.WalkableAt(rightRear) {
				ntt.direction = int2dir(grid.RNG().Intn(4))
				ntt.state = mstToWall
			} else if grid.PassableAt(ahead) && !grid.PassableAt(right) {
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
			} else if grid.PassableAt(right) && !grid.PassableAt(rightRear) {
				ntt.TurnRight()
				return stay
			} else {
				ntt.direction = int2dir(grid.RNG().Intn(4))
				ntt.state = mstToWall
			}
		}
	}
	return ntt.Location
}
func (ntt *Monster) CanDamage(other Entity) bool {
	return other.IsPlayer()
}
func (ntt *Monster) CollisionFrom(other Entity) {
	other.SetCollided()
	if other.CanDamage(ntt) {
		ntt.ChangeHealth(-2)
		other.AddMessage("hit monster, 2 damage")
	}
}
func (ntt *Monster) DeathSpawn() (Entity, bool) {
	return NewLoot(), true
}
func (ntt *Monster) Detect(player Entity) {
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
func (ntt *Monster) IsDead() bool {
	return ntt.Health() <= 0
}

type Loot struct {
	EntityT
}

func NewLoot() Entity {
	ntt := EntityT{
		ID:     NewEntityID(),
		Symbol: '+',
	}
	newLoot := Loot{
		EntityT: ntt,
	}
	return &newLoot
}

func (ntt *Loot) CollisionFrom(other Entity) {
	if other.IsPlayer() {
		other.ChangeHealth(6)
		other.AddMessage("heal +6")
	}
}

func (ntt *Loot) IsDead() bool {
	return false
}

func (ntt *Loot) IsWalkable() bool {
	return true
}
