package main

import (
	"bytes"
	//"fmt"
	"math/rand"
	"strconv"
	"time"
)

type GridProcessor interface {
	TickNumber() uint64
}

type GridKeeper interface {
	DeferMove(Creature, Coord)
	DungeonAt(Coord) int8
	EmptyAt(Coord) bool
	MoveEntity(Creature, Coord)
	NewEntity(Creature) (Creature, bool)
	OutOfBounds(Coord) bool
	PutEntityAt(Creature, Coord)
	RemoveEntityID(EntityID)
	UpdateMovers(GridProcessor)
	SendDisplays(GridProcessor)
	WalkableAt(Coord) bool
	WriteDisplay(Creature, *bytes.Buffer)
	WriteEntities(Creature, *bytes.Buffer)
}

func ExecuteMove(ntt Creature, grid GridKeeper, loc Coord) {
	if grid.OutOfBounds(loc) {
		grid.DeferMove(ntt, loc)
	} else if grid.EmptyAt(loc) && grid.WalkableAt(loc) {
		LogTrace("about to move", ntt, loc)
		grid.MoveEntity(ntt, loc)
	}
}

type DeferredMove struct {
	id  EntityID
	loc Coord
}

type SubGrid struct {
	dunGenCache *DunGenCache
	GridCoord   GridCoord
	Grid        map[Coord]EntityID
	Entities    map[EntityID]Creature
	parent      *WorldGrid
	ParentQueue chan DeferredMove
	PlayerCount int
	RNG         *rand.Rand
}

func NewSubGrid(gcoord GridCoord) *SubGrid {
	return &SubGrid{
		dunGenCache: NewDunGenCache(10, DungeonEntropy, DungeonProto),
		GridCoord:   gcoord,
		Grid:        make(map[Coord]EntityID),
		Entities:    make(map[EntityID]Creature),
		ParentQueue: make(chan DeferredMove, ((2 * subgrid_width) + (2 * subgrid_height))),
		RNG:         rand.New(rand.NewSource(time.Now().UnixNano())),
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
func (self *SubGrid) DeferMove(ntt Creature, loc Coord) {
	deferredMove := DeferredMove{
		id:  ntt.EntityID(),
		loc: loc,
	}
	self.ParentQueue <- deferredMove
}
func (self *SubGrid) EmptyAt(loc Coord) bool {
	if loc.Grid() != self.GridCoord {
		ERROR.Panic("Should not be asked about outside coord!")
	}
	_, present := self.Grid[loc]
	return !present
}

const subgrid_placement_trys = 100

func (self *SubGrid) MoveEntity(ntt Creature, loc Coord) {
	if ntt.Coord() != loc {
		if loc.Grid() != self.GridCoord {
			ERROR.Panic("Should not be here!")
		} else {
			delete(self.Grid, ntt.Coord())
			self.Grid[loc] = ntt.EntityID()
			ntt.SetCoord(loc)
		}
	}
}
func (self *SubGrid) RandomCoord() Coord {
	lx := self.RNG.Intn(subgrid_width)
	ly := self.RNG.Intn(subgrid_height)
	x := (self.GridCoord.x * subgrid_width) + int64(lx)
	y := (self.GridCoord.y * subgrid_height) + int64(ly)
	return Coord{x, y}
}

func (self *SubGrid) NewEntity(ntt Creature) (Creature, bool) {
	loc := self.RandomCoord()
	for n := 0; (!(self.EmptyAt(loc) && self.WalkableAt(loc))) && (n < subgrid_placement_trys); n++ {
		loc = self.RandomCoord()
	}
	if !self.EmptyAt(loc) {
		return &Entity{}, false
	}
	ntt.SetCoord(loc)
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
func (self *SubGrid) PutEntityAt(ntt Creature, loc Coord) {
	if loc.Grid() != self.GridCoord {
		ERROR.Panic("Should not put outside coord!")
	}
	ntt.SetCoord(loc)
	self.Grid[loc] = ntt.EntityID()
	self.Entities[ntt.EntityID()] = ntt
	if ntt.IsPlayer() {
		self.PlayerCount++
	}
}
func (self *SubGrid) RemoveEntityID(id EntityID) {
	ntt, present := self.Entities[id]
	if !present {
		ERROR.Panic("Removing nonexistent Entity")
	}
	if ntt.IsPlayer() {
		self.PlayerCount--
	}
	delete(self.Grid, ntt.Coord())
	delete(self.Entities, id)
}
func (self *SubGrid) WriteEntities(player Creature, buffer *bytes.Buffer) {
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
	}
}
func (self *SubGrid) SendDisplays(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		ntt.SendDisplay(self, gproc)
	}
}

// WriteDisplay can only be called on the SubGrid through ParallelExec()
// It is not concurrent
func (self *SubGrid) WriteDisplay(ntt Creature, buffer *bytes.Buffer) {
	x, y := ntt.Coord().x, ntt.Coord().y
	buffer.WriteString(`{"type":"update","data":{`)
	buffer.WriteString(`"location":[`)
	buffer.WriteString(strconv.FormatInt(x, 10))
	buffer.WriteRune(',')
	buffer.WriteString(strconv.FormatInt(y, 10))
	buffer.WriteString(`],`)
	self.dunGenCache.WriteMap(ntt, buffer)
	buffer.WriteRune(',')
	buffer.WriteString(`"entities":{`)

	// This call to parent works concurrently. It's read only.
	// It has to be the parent to coordinate all visible SubGrids
	self.parent.WriteEntities(ntt, buffer)

	buffer.WriteString(`}}}`)
}

type WorldGrid struct {
	dunGenCache *DunGenCache
	grid        map[GridCoord]*SubGrid
	entityGrid  map[EntityID]GridCoord
	RNG         *rand.Rand
	spawnGrids  []GridCoord
}

func NewWorldGrid() *WorldGrid {
	spawnGrids := []GridCoord{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {-1, -1}, {-1, 0}, {0, -1}}
	return &WorldGrid{
		dunGenCache: NewDunGenCache(1000, DungeonEntropy, DungeonProto),
		grid:        make(map[GridCoord]*SubGrid),
		entityGrid:  make(map[EntityID]GridCoord),
		RNG:         rand.New(rand.NewSource(time.Now().UnixNano())),
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
		if subgrid.Count() == 0 {
			delete(self.grid, gc)
		}
	}
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
	expand1 := expandGrids(pgrids)
	prepop := expandGrids(expand1)
	actual := self.actualGridCoord()
	cull := copyGrids(actual)
	subtractGrids(cull, pgrids)
	subtractGrids(cull, prepop)
	subtractGrids(prepop, actual)
	subtractGrids(prepop, expand1)
	subtractGridList(prepop, self.spawnGrids)
	return prepop, cull
}

func (self *WorldGrid) prepopulateGrids(grids *(map[GridCoord]bool)) {
	if len(*grids) <= 0 {
		return
	}
	n := self.RNG.Intn(len(*grids))
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

func (self *WorldGrid) WriteEntities(player Creature, buffer *bytes.Buffer) {
	coord := player.Coord()
	visibleGrids := coord.VisibleGrids(39, 12)
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

func (self *WorldGrid) DeferMove(ntt Creature, loc Coord) {}
func (self *WorldGrid) EmptyAt(loc Coord) bool {
	subgrid, present := self.grid[loc.Grid()]
	if !present {
		return true
	}
	return subgrid.EmptyAt(loc)
}
func (self *WorldGrid) MoveEntity(ntt Creature, loc Coord) {
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
func (self *WorldGrid) NewEntity(ntt Creature) (Creature, bool) {
	var newEntity Creature
	ntt.SetEntityID(NewEntityID())
	ok := false
	for !ok {
		i := self.RNG.Intn(len(self.spawnGrids))
		gridCoord := self.spawnGrids[i]
		subgrid := self.subgridAtGrid(gridCoord)
		newEntity, ok = subgrid.NewEntity(ntt)
		if ok {
			self.entityGrid[ntt.EntityID()] = gridCoord
		}
	}
	return newEntity, ok
}
func (self *WorldGrid) NewEntityInGrid(ntt Creature, gridCoord GridCoord) (Creature, bool) {
	var newEntity Creature
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
func (self *WorldGrid) PutEntityAt(ntt Creature, loc Coord) {
	_, present := self.entityGrid[ntt.EntityID()]
	if present {
		ERROR.Panic("Placing already existing Entity")
	}
	gridCoord := loc.Grid()
	self.entityGrid[ntt.EntityID()] = gridCoord
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.PutEntityAt(ntt, loc)
}
func (self *WorldGrid) RemoveEntityID(id EntityID) {
	gridCoord, present := self.entityGrid[id]
	if !present {
		ERROR.Panic("Removing nonexistent Entity")
	}
	delete(self.entityGrid, id)
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.RemoveEntityID(id)
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
				ExecuteMove(ntt, self, deferred.loc)
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
func (self *WorldGrid) WriteDisplay(ntt Creature, buffer *bytes.Buffer) {
	ERROR.Panic("Should not call on World!")
}
