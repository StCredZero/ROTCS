package main

import (
	"bytes"
	"fmt"
	//"math"
	"html/template"
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
	ClearLActToggle()
	ClearLifeToggle()
	Collided() bool
	CollideWall()
	CollisionFrom(other Entity)
	Coord() Coord
	DeathSpawn() (Entity, bool)
	Detect(Entity)
	Direction() rune
	DisplayString() string
	FormattedMessage(string) string
	GetSubgrid() *SubGrid
	HasMove(GridProcessor) bool
	Health() int
	Inbox() []string
	Initialized() bool
	InMaxRange(Entity) bool
	IsBlurred() bool
	IsDead() bool
	IsPlayer() bool
	IsTransient() bool
	IsWalkable() bool
	LastDispCoord() Coord
	LActToggle() bool
	LifeToggle() bool
	MoveCommit()
	MoveTimestamp() uint64
	Outbox() []string
	SendDisplay(GridKeeper, GridProcessor)
	SetCoord(Coord)
	SetCollided()
	SetEntityID(EntityID)
	SetHealth(int)
	SetInitialized(bool)
	SetLActToggle()
	SetLifeToggle()
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
func (ntt *EntityT) ClearLActToggle() {}
func (ntt *EntityT) ClearLifeToggle() {}
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
func (ntt *EntityT) Direction() rune {
	return ntt.direction
}
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
func (ntt *EntityT) InMaxRange(other Entity) bool {
	return ntt.Location.InMaxRange(other.Coord())
}
func (ntt *EntityT) IsBlurred() bool   { return false }
func (ntt *EntityT) IsDead() bool      { return true }
func (ntt *EntityT) IsPlayer() bool    { return false }
func (ntt *EntityT) IsTransient() bool { return true }
func (ntt *EntityT) IsWalkable() bool  { return false }
func (ntt *EntityT) LastDispCoord() Coord {
	return ntt.Location
}
func (ntt *EntityT) LActToggle() bool { return false }
func (ntt *EntityT) LifeToggle() bool { return false }
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
func (ntt *EntityT) MoveCommit() {}
func (ntt *EntityT) MoveTimestamp() uint64 {
	return 0
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
func (ntt *EntityT) SetLActToggle() {}
func (ntt *EntityT) SetLifeToggle() {}
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
	buffer.WriteRune(self.Symbol)
}

type Player struct {
	EntityT
	collided      bool
	Connection    *connection
	inbox         []string
	LastUpdateLoc Coord
	lactToggle    bool
	lifeToggle    bool
	moveBuffer    []moveRequest
	moveQueue     chan moveRequest
	moveTimestamp uint64
	outbox        []string
	outQueue      chan string
}

func NewPlayer(c *connection) *Player {
	entity := EntityT{
		direction:    '0',
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
		moveBuffer: make([]moveRequest, 0, 4),
		moveQueue:  make(chan moveRequest, 64),
		outbox:     make([]string, 0, 20),
		outQueue:   make(chan string, 256),
	}
}

func (ntt *Player) AddMessage(msg string) {
	ntt.inbox = append(ntt.inbox, msg)
}
func (ntt *Player) CalcMove(grid GridKeeper) Coord {
	select {
	case mv := <-ntt.moveQueue:
		ntt.moveBuffer = append(make([]moveRequest, 0, 4), mv)
	default:
	}
moveqloop:
	for {
		select {
		case mv := <-ntt.moveQueue:
			ntt.moveBuffer = append(ntt.moveBuffer, mv)
		default:
			break moveqloop
		}
	}
	if len(ntt.moveBuffer) > 0 {
		move := ntt.moveBuffer[0]
		ntt.moveTimestamp = move.timestamp
		return ntt.Location.MovedBy(move.direction)
	}
	return ntt.Location
}
func (ntt *Player) CanDamage(other Entity) bool {
	return !other.IsPlayer()
}
func (ntt *Player) CanSwapWith(other Entity) bool {
	return other.IsPlayer()
}
func (ntt *Player) ClearLActToggle() {
	ntt.lactToggle = false
}
func (ntt *Player) ClearLifeToggle() {
	ntt.lifeToggle = false
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
			ntt.ChangeHealth(-3)
			ntt.AddMessage("flank hit, damage -3")
		}
	}
}
func (ntt *Player) Detect(player Entity) {
	//if player.IsPlayer() {
	var buffer bytes.Buffer
	for _, message := range player.Outbox() {
		safe := template.HTMLEscapeString(message)
		buffer.WriteString(`{"type":"message","data":"`)
		buffer.WriteString(player.FormattedMessage(safe))
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
func (ntt *Player) IsBlurred() bool {
	return ntt.Connection.IsBlurred
}
func (ntt *Player) IsDead() bool {
	return ntt.Health() <= -10
}
func (ntt *Player) IsPlayer() bool    { return true }
func (ntt *Player) IsTransient() bool { return false }
func (ntt *Player) LActToggle() bool {
	return ntt.lactToggle
}
func (ntt *Player) LifeToggle() bool {
	return ntt.lifeToggle
}
func (ntt *Player) LastDispCoord() Coord { return ntt.LastUpdateLoc }
func (ntt *Player) MoveCommit() {
	if len(ntt.moveBuffer) > 0 {
		ntt.direction = (ntt.moveBuffer[0]).direction
		ntt.moveBuffer = ntt.moveBuffer[1:]
	}
}
func (ntt *Player) MoveTimestamp() uint64 {
	return ntt.moveTimestamp
}
func (ntt *Player) Outbox() []string {
	return ntt.outbox
}
func (ntt *Player) SendDisplay(grid GridKeeper, gproc GridProcessor) {
	LogTrace("Start SendDisplay ", ntt.Location)
	if ntt.IsBlurred() {
		LogTrace("No SendDisplay: Blurred ", ntt.Location)
		return
	}
	var buffer bytes.Buffer
	grid.WriteDisplay(ntt, gproc, &buffer)
	ntt.Connection.send <- buffer.Bytes()
	ntt.LastUpdateLoc = ntt.Location
	ntt.collided = false
	ntt.inbox = ntt.inbox[:0]
	ntt.outbox = ntt.outbox[:0]
	LogTrace("End SendDisplay ", ntt.Location)
}
func (ntt *Player) SetLActToggle() {
	ntt.lactToggle = true
}
func (ntt *Player) SetLifeToggle() {
	ntt.lifeToggle = true
}

type detection struct {
	id   EntityID
	loc  Coord
	dist float64
}

type Monster struct {
	EntityT
	detections  []detection
	detectQueue chan detection
	state       int
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
		EntityT:     entity,
		detections:  make([]detection, 0, 4),
		detectQueue: make(chan detection, (subgrid_width * subgrid_height)),
		state:       mstStart,
	}
}

func (ntt *Monster) CalcMove(grid GridKeeper) Coord {
detectqloop:
	for {
		select {
		case det := <-ntt.detectQueue:
			ntt.detections = append(ntt.detections, det)
		default:
			break detectqloop
		}
	}
	min := detection{
		dist: 1000000.0,
	}
	minFound := false
	for _, det := range ntt.detections {
		if det.dist < min.dist {
			min = det
			minFound = true
		}
	}
	if minFound {
		ntt.state = mstStart
		openAt := func(coord Coord) bool {
			return grid.WalkableAt(coord)
		}
		path, pathFound := astarSearch(distance, openAt, neighbors4, ntt.Coord(), min.loc, 200)
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
				ntt.state = mstStart
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
	if player.IsPlayer() {
		loc1, loc2 := ntt.Coord(), player.Coord()
		dist := distance(loc1, loc2)
		if dist <= 7 {
			det := detection{
				id:   player.EntityID(),
				loc:  loc2,
				dist: dist,
			}
			ntt.detectQueue <- det
		}
	}
}
func (ntt *Monster) DisplayString() string {
	return "Exo"
}
func (ntt *Monster) IsDead() bool {
	return ntt.Health() <= 0
}
func (ntt *Monster) IsPlayer() bool {
	return false
}

func (ntt *Monster) MoveCommit() {
	ntt.detections = make([]detection, 0, 4)
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
		other.ChangeHealth(2)
		other.AddMessage("heal +2")
	}
}

func (ntt *Loot) IsDead() bool {
	return false
}

func (ntt *Loot) IsWalkable() bool {
	return true
}

type ShipGuard struct {
	EntityT
}

func NewShipGuard() Entity {
	ntt := EntityT{
		ID:     NewEntityID(),
		Symbol: 'G',
	}
	newNtt := ShipGuard{
		EntityT: ntt,
	}
	return &newNtt
}

func (ntt *ShipGuard) CollisionFrom(other Entity) {
	if other.IsPlayer() {
		loc1 := ntt.Coord()
		loc2 := other.Coord()
		loc3 := Coord{loc1.x + (loc1.x - loc2.x), loc1.y + (loc1.y - loc2.y)}
		if loc2.Grid() == loc3.Grid() {
			atLoc3, present := ntt.subgrid.EntityAt(loc3)
			if present {
				if atLoc3.IsPlayer() {
					ntt.subgrid.SwapEntities(other, atLoc3)
				}
			} else {
				ntt.subgrid.MoveEntity(other, loc3)
			}
		}
	}
}

func (ntt *ShipGuard) IsDead() bool {
	return false
}
func (ntt *ShipGuard) IsTransient() bool {
	return false
}
