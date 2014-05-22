package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

type GridProcessor interface {
	ServerLoad() float64
	ServerPopulation() int
	TickNumber() uint64
}

type GridKeeper interface {
	DeferMove(Entity, Coord)
	DungeonAt(Coord) int8
	EmptyAt(Coord) bool
	EntityAt(Coord) (Entity, bool)
	EntityByID(EntityID) Entity
	MarkDead(Entity)
	MoveEntity(Entity, Coord)
	NewEntity(Entity) (Entity, bool)
	OutOfBounds(Coord) bool
	PassableAt(Coord) bool
	PutEntityAt(Entity, Coord)
	RemoveEntityID(EntityID)
	ReplaceEntity(Entity, Entity)
	RNG() *rand.Rand
	SendDisplays(GridProcessor)
	SwapEntities(Entity, Entity)
	UpdateMovers(GridProcessor)
	WalkableAt(Coord) bool
	WriteDisplay(Entity, GridProcessor, *bytes.Buffer)
	WriteEntities(Entity, *bytes.Buffer)
}

func ExecuteMove(ntt Entity, grid GridKeeper, loc Coord) {
	if grid.OutOfBounds(loc) {
		grid.DeferMove(ntt, loc)
		return
	}
	if grid.WalkableAt(loc) {
		other, present := grid.EntityAt(loc)
		if present {
			if ntt.CanSwapWith(other) {
				grid.SwapEntities(ntt, other)
			} else if other.IsWalkable() {
				other.CollisionFrom(ntt)
				grid.ReplaceEntity(other, ntt)
			} else {
				other.CollisionFrom(ntt)
			}
		} else {
			grid.MoveEntity(ntt, loc)
		}
	} else {
		ntt.CollideWall()
	}
	ntt.MoveCommit()
}

type DeferredMove struct {
	id  EntityID
	loc Coord
}

type SubGrid struct {
	chatQueue   chan string
	deaths      []EntityID
	dunGenCache *DunGenCache
	GridCoord   GridCoord
	Grid        map[Coord]EntityID
	Entities    map[EntityID]Entity
	parent      *WorldGrid
	ParentQueue chan DeferredMove
	PlayerCount int
	rng         *rand.Rand
}

func NewSubGrid(gcoord GridCoord) *SubGrid {
	return &SubGrid{
		chatQueue:   make(chan string, (subgrid_width * subgrid_height)),
		deaths:      make([]EntityID, 0, 10),
		dunGenCache: NewDunGenCache(10, DungeonEntropy, DungeonProto),
		GridCoord:   gcoord,
		Grid:        make(map[Coord]EntityID),
		Entities:    make(map[EntityID]Entity),
		ParentQueue: make(chan DeferredMove, ((2 * subgrid_width) + (2 * subgrid_height))),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (self *SubGrid) DungeonAt(coord Coord) int8 {
	return self.dunGenCache.DungeonAt(coord)
}

func (srv *SubGrid) WalkableAt(coord Coord) bool {
	return srv.dunGenCache.WalkableAt(coord)
}

func (self *SubGrid) Count() int {
	return len(self.Grid)
}
func (self *SubGrid) DeferMove(ntt Entity, loc Coord) {
	deferredMove := DeferredMove{
		id:  ntt.EntityID(),
		loc: loc,
	}
	self.ParentQueue <- deferredMove
}
func (self *SubGrid) EmptyAt(loc Coord) bool {
	_, present := self.Grid[loc]
	return !present
}
func (self *SubGrid) EntityAt(loc Coord) (Entity, bool) {
	id, present := self.Grid[loc]
	if present {
		return self.Entities[id], true
	} else {
		return nil, false
	}
}
func (self *SubGrid) EntityByID(id EntityID) Entity {
	ntt, present := self.Entities[id]
	if present {
		return ntt
	} else {
		return nil
	}
}
func (self *SubGrid) MarkDead(ntt Entity) {
	self.deaths = append(self.deaths, ntt.EntityID())
}

const subgrid_placement_trys = 100

func (self *SubGrid) MoveEntity(ntt Entity, loc Coord) {
	if ntt.Coord() != loc {
		if loc.Grid() != self.GridCoord {
			ERROR.Panic(`Should not be here!`)
		} else {
			delete(self.Grid, ntt.Coord())
			self.Grid[loc] = ntt.EntityID()
			ntt.SetCoord(loc)
		}
	}
}
func (self *SubGrid) RandomCoord() Coord {
	lx := self.rng.Intn(subgrid_width)
	ly := self.rng.Intn(subgrid_height)
	x := (self.GridCoord.x * subgrid_width) + int64(lx)
	y := (self.GridCoord.y * subgrid_height) + int64(ly)
	return Coord{x, y}
}

func (self *SubGrid) NewEntity(ntt Entity) (Entity, bool) {
	loc := self.RandomCoord()
	for n := 0; (!(self.EmptyAt(loc) && self.WalkableAt(loc))) && (n < subgrid_placement_trys); n++ {
		loc = self.RandomCoord()
	}
	if !self.EmptyAt(loc) {
		return nil, false
	}
	ntt.SetCoord(loc)
	ntt.SetSubgrid(self)
	self.Entities[ntt.EntityID()] = ntt
	self.Grid[loc] = ntt.EntityID()
	if ntt.IsPlayer() {
		self.PlayerCount++
	}
	return ntt, true
}
func (self *SubGrid) OutOfBounds(coord Coord) bool {
	return (coord.Grid() != self.GridCoord)
}
func (self *SubGrid) PassableAt(loc Coord) bool {
	return self.EmptyAt(loc) && self.WalkableAt(loc)
}
func (self *SubGrid) PutEntityAt(ntt Entity, loc Coord) {
	if loc.Grid() != self.GridCoord {
		ERROR.Panic("Should not put outside coord!")
	}
	ntt.SetCoord(loc)
	ntt.SetSubgrid(self)
	self.Grid[loc] = ntt.EntityID()
	self.Entities[ntt.EntityID()] = ntt
	if ntt.IsPlayer() {
		self.PlayerCount++
	}
}
func (self *SubGrid) RemoveEntityID(id EntityID) {
	ntt, present := self.Entities[id]
	if !present {
		return
	}
	if ntt.IsPlayer() {
		self.PlayerCount--
	}
	ntt.SetSubgrid(nil)
	delete(self.Grid, ntt.Coord())
	delete(self.Entities, id)
}
func (self *SubGrid) ReplaceEntity(ntt, replacement Entity) {
	loc := ntt.Coord()
	self.RemoveEntityID(ntt.EntityID())
	self.MoveEntity(replacement, loc)
}
func (self *SubGrid) RNG() *rand.Rand {
	return self.rng
}
func (self *SubGrid) SwapEntities(ntt, other Entity) {
	nttLoc, otherLoc := ntt.Coord(), other.Coord()
	grid1, grid2 := nttLoc.Grid(), otherLoc.Grid()
	if grid1 != self.GridCoord || grid2 != self.GridCoord {
		ERROR.Panic(`Subgrid swapping outside bounds!`)
	}
	ntt.SetCoord(otherLoc)
	other.SetCoord(nttLoc)
	self.Grid[nttLoc] = other.EntityID()
	self.Grid[otherLoc] = ntt.EntityID()
}
func (self *SubGrid) WriteEntities(player Entity, buffer *bytes.Buffer) {
	for _, id := range self.Grid {
		if id != player.EntityID() {
			ntt := self.Entities[id]
			ntt.WriteFor(player, buffer)
			ntt.Detect(player)
			buffer.WriteString(`,`)
		}
	}
}
func (self *SubGrid) UpdateMovers(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		if ntt.HasMove(gproc) {
			loc := ntt.CalcMove(self)
			ExecuteMove(ntt, self, loc)
		}
		if ntt.IsDead() {
			self.MarkDead(ntt)
		}
	}
}
func (self *SubGrid) SendDisplays(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		ntt.SendDisplay(self, gproc)
	}
}

// WriteDisplay can only be called on the SubGrid through ParallelExec()
// It is not concurrent
func (self *SubGrid) WriteDisplay(ntt Entity, gproc GridProcessor, buffer *bytes.Buffer) {
	x, y := ntt.Coord().x, ntt.Coord().y
	buffer.WriteString(`{"type":"update",`)

	buffer.WriteString(`"pop":`)
	buffer.WriteString(strconv.FormatInt(int64(gproc.ServerPopulation()), 10))
	buffer.WriteRune(',')

	buffer.WriteString(`"load":`)
	buffer.WriteString(strconv.FormatFloat(gproc.ServerLoad(), 'f', 2, 64))
	buffer.WriteRune(',')

	buffer.WriteString(`"location":[`)
	buffer.WriteString(strconv.FormatInt(x, 10))
	buffer.WriteRune(',')
	buffer.WriteString(strconv.FormatInt(y, 10))
	buffer.WriteString(`],`)

	buffer.WriteString(`"health":`)
	buffer.WriteString(strconv.FormatInt(int64(ntt.Health()), 10))
	buffer.WriteRune(',')

	if gproc.TickNumber()%23 == 0 {
		self.dunGenCache.WriteBasicMap(ntt, buffer)
	} else {
		self.dunGenCache.WriteMap(ntt, buffer)
	}

	buffer.WriteRune(',')
	buffer.WriteString(`"entities":{`)

	// This call to parent works concurrently. It's read only.
	// It has to be the parent to coordinate all visible SubGrids
	self.parent.WriteEntities(ntt, buffer)

	buffer.WriteString(`},`)
	buffer.WriteString(`"collided":`)
	buffer.WriteRune(bool2rune(ntt.Collided()))
	buffer.WriteRune(',')
	buffer.WriteString(`"messages":[`)
	for _, msg := range ntt.Inbox() {
		buffer.WriteRune('"')
		buffer.WriteString(msg)
		buffer.WriteString(`",`)
	}
	buffer.WriteString(`""],`)

	buffer.WriteString(`"timestamp":`)
	buffer.WriteString(strconv.FormatUint(ntt.MoveTimestamp(), 10))
	buffer.WriteRune('}')
}

func (self *SubGrid) SendMessages() {

}

type WorldGrid struct {
	deaths      []EntityID
	dunGenCache *DunGenCache
	grid        map[GridCoord]*SubGrid
	entityGrid  map[EntityID]GridCoord
	rng         *rand.Rand
	spawnGrids  []GridCoord
}

func NewWorldGrid() *WorldGrid {
	spawnGrids := []GridCoord{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {-1, -1}, {-1, 0}, {0, -1}}
	dgCache := NewDunGenCache(1000, DungeonEntropy, DungeonProto)

	return &WorldGrid{
		dunGenCache: dgCache,
		grid:        make(map[GridCoord]*SubGrid),
		entityGrid:  make(map[EntityID]GridCoord),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		spawnGrids:  spawnGrids,
	}
}

func (self *WorldGrid) subgridAtGrid(gridCoord GridCoord) *SubGrid {
	subgrid, present := self.grid[gridCoord]
	if !present {
		subgrid = NewSubGrid(gridCoord)
		subgrid.parent = self
		self.grid[gridCoord] = subgrid
	}
	return subgrid
}

func (self *WorldGrid) playerCount() int {
	count := 0
	for _, subgrid := range self.grid {
		count += subgrid.PlayerCount
	}
	return count
}

func (self *WorldGrid) playerGrids() *(map[GridCoord]bool) {
	grids := make(map[GridCoord]bool)
	for _, subgrid := range self.grid {
		if subgrid.PlayerCount > 0 {
			grids[subgrid.GridCoord] = true
		}
	}
	return &grids
}

func (self *WorldGrid) discardEmpty() {
	for gc, subgrid := range self.grid {
		for _, id := range subgrid.deaths {
			dead := self.EntityByID(id)
			loc := dead.Coord()
			spawn, spawned := dead.DeathSpawn()
			self.RemoveEntityID(id)
			if spawned {
				self.PutEntityAt(spawn, loc)
			}
		}
		subgrid.deaths = subgrid.deaths[:0]
		if subgrid.Count() == 0 {
			delete(self.grid, gc)
		}
	}
	for _, id := range self.deaths {
		dead := self.EntityByID(id)
		loc := dead.Coord()
		spawn, spawned := dead.DeathSpawn()
		self.RemoveEntityID(id)
		if spawned {
			self.PutEntityAt(spawn, loc)
		}
	}
	self.deaths = self.deaths[:0]
}

func (self *WorldGrid) actualGridCoord() *(map[GridCoord]bool) {
	grids := make(map[GridCoord]bool)
	for gc, _ := range self.grid {
		grids[gc] = true
	}
	return &grids
}

func (self *WorldGrid) prepopCullGrids() (*(map[GridCoord]bool), *(map[GridCoord]bool)) {
	pgrids := self.playerGrids()
	(*pgrids)[GridCoord{0, 0}] = true
	expand1 := expandGrids(pgrids)
	prepop := expandGrids(expand1)
	actual := self.actualGridCoord()
	cull := copyGrids(actual)

	subtractGrids(cull, pgrids)
	subtractGrids(cull, prepop)
	subtractGridList(cull, self.spawnGrids)

	subtractGrids(prepop, actual)
	subtractGrids(prepop, expand1)
	subtractGridList(prepop, self.spawnGrids)

	return prepop, cull
}

func (self *WorldGrid) prepopulateGrids(grids *(map[GridCoord]bool)) {
	if len(*grids) <= 0 {
		return
	}
	n := self.rng.Intn(len(*grids))
	i := 0
	for gcoord, _ := range *grids {
		if i == n {
			ok := true
			for tries := 0; ok && tries < 10; tries++ {
				monster := NewMonster(NewEntityID())
				_, ok = self.NewEntityInGrid(monster, gcoord)
			}
			break
		}
		i++
	}
}

func (self *WorldGrid) cullGrids(grids *(map[GridCoord]bool)) {
	for gcoord, _ := range *grids {
		subgrid := self.subgridAtGrid(gcoord)
		for id, ntt := range subgrid.Entities {
			if ntt.IsTransient() {
				subgrid.RemoveEntityID(id)
			}
		}
		break
	}
}

func (self *WorldGrid) WriteEntities(player Entity, buffer *bytes.Buffer) {
	coord := player.Coord()
	var gcoords [4]GridCoord
	visibleGrids := coord.VisibleGrids(39, 12, gcoords[:])
	for _, gcoord := range visibleGrids {
		subgrid, present := self.grid[gcoord]
		if present {
			subgrid.WriteEntities(player, buffer)
		}
	}
	buffer.WriteString(`"e":""`)
}
func (self *WorldGrid) DungeonAt(coord Coord) int8 {
	return self.dunGenCache.DungeonAt(coord)
}

func (srv *WorldGrid) WalkableAt(coord Coord) bool {
	return srv.dunGenCache.WalkableAt(coord)
}

func (self *WorldGrid) DeferMove(ntt Entity, loc Coord) {}
func (self *WorldGrid) EmptyAt(loc Coord) bool {
	subgrid, present := self.grid[loc.Grid()]
	if !present {
		return true
	}
	return subgrid.EmptyAt(loc)
}
func (self *WorldGrid) EntityAt(loc Coord) (Entity, bool) {
	gridCoord := loc.Grid()
	subgrid, present := self.grid[gridCoord]
	if !present {
		return nil, false
	} else {
		ntt, present := subgrid.EntityAt(loc)
		return ntt, present
	}
}
func (self *WorldGrid) EntityByID(id EntityID) Entity {
	gc, present := self.entityGrid[id]
	if present {
		sg := self.subgridAtGrid(gc)
		return sg.EntityByID(id)
	} else {
		return nil
	}
}

func (self *WorldGrid) MarkDead(ntt Entity) {
	self.deaths = append(self.deaths, ntt.EntityID())
}
func (self *WorldGrid) MoveEntity(ntt Entity, loc Coord) {
	gc1, present := self.entityGrid[ntt.EntityID()]
	if !present {
		ERROR.Panic("Moving nonexistent Entity")
	}
	sg1 := self.subgridAtGrid(gc1)
	gc2 := loc.Grid()
	if gc1 == gc2 {
		sg1.MoveEntity(ntt, loc)
	} else {
		sg2 := self.subgridAtGrid(gc2)
		sg1.RemoveEntityID(ntt.EntityID())
		sg2.PutEntityAt(ntt, loc)
		self.entityGrid[ntt.EntityID()] = gc2
	}
}
func (self *WorldGrid) NewEntity(ntt Entity) (Entity, bool) {
	var newEntity Entity
	ntt.SetEntityID(NewEntityID())
	ok := false
	for !ok {
		for i := 0; !ok && i < len(self.spawnGrids); i++ {
			gridCoord := self.spawnGrids[i]
			subgrid := self.subgridAtGrid(gridCoord)
			newEntity, ok = subgrid.NewEntity(ntt)
			if ok {
				self.entityGrid[ntt.EntityID()] = gridCoord
			}
		}
	}
	return newEntity, ok
}
func (self *WorldGrid) NewEntityInGrid(ntt Entity, gridCoord GridCoord) (Entity, bool) {
	var newEntity Entity
	ntt.SetEntityID(NewEntityID())
	done := false
	subgrid := self.subgridAtGrid(gridCoord)
	for tries := 0; !done && tries < 50; tries++ {
		newEntity, done = subgrid.NewEntity(ntt)
	}
	if done {
		self.entityGrid[ntt.EntityID()] = gridCoord
	}
	return newEntity, done
}
func (self *WorldGrid) OutOfBounds(coord Coord) bool { return false }
func (self *WorldGrid) PassableAt(loc Coord) bool {
	return self.EmptyAt(loc) && self.WalkableAt(loc)
}
func (self *WorldGrid) PutEntityAt(ntt Entity, loc Coord) {
	_, present := self.entityGrid[ntt.EntityID()]
	if present {
		ERROR.Panic("Placing already existing Entity")
	}
	gridCoord := loc.Grid()
	self.entityGrid[ntt.EntityID()] = gridCoord
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.PutEntityAt(ntt, loc)
}
func (self *WorldGrid) RNG() *rand.Rand {
	return self.rng
}
func (self *WorldGrid) RemoveEntityID(id EntityID) {
	gridCoord, present := self.entityGrid[id]
	if !present {
		return
	}
	delete(self.entityGrid, id)
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.RemoveEntityID(id)
}
func (self *WorldGrid) ReplaceEntity(ntt, replacement Entity) {
	loc := ntt.Coord()
	self.RemoveEntityID(ntt.EntityID())
	self.MoveEntity(replacement, loc)
}
func (self *WorldGrid) SwapEntities(ntt, other Entity) {
	nttLoc, otherLoc := ntt.Coord(), other.Coord()
	self.RemoveEntityID(ntt.EntityID())
	self.RemoveEntityID(other.EntityID())
	self.PutEntityAt(ntt, otherLoc)
	self.PutEntityAt(other, nttLoc)
}
func (self *WorldGrid) ParallelExec(doWork func(*SubGrid)) {
	n := len(self.grid)
	semaphore := make(chan empty, n)
	for _, subgrid := range self.grid {
		go func(sg *SubGrid) {
			doWork(sg)
			semaphore <- empty{}
		}(subgrid)
	}
	for i := 0; i < n; i++ {
		<-semaphore
	}
}
func (self *WorldGrid) UpdateMovers(gproc GridProcessor) {
	self.ParallelExec(func(subgrid *SubGrid) {
		subgrid.UpdateMovers(gproc)
	})
	for _, subgrid := range self.grid {
		done := false
		for !done {
			select {
			case deferred := <-subgrid.ParentQueue:
				ntt := subgrid.Entities[deferred.id]
				loc := ntt.CalcMove(self)
				ExecuteMove(ntt, self, loc)
				if ntt.IsDead() {
					self.MarkDead(ntt)
				}
			default:
				done = true
			}
		}
	}
}
func (self *WorldGrid) SendDisplays(gproc GridProcessor) {
	self.ParallelExec(func(subgrid *SubGrid) {
		subgrid.SendDisplays(gproc)
	})
}
func (self *WorldGrid) WriteDisplay(ntt Entity, gproc GridProcessor, buffer *bytes.Buffer) {
	ERROR.Panic("Should not call on World!")
}
