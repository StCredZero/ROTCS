package main

import (
	"bytes"
	//"fmt"
	"math/rand"
	"strconv"
	"sync"
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
	LifeActive() bool
	LifeGridAt(Coord) bool
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
	SetLifeGridAt(Coord, bool)
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
	if grid.LifeActive() {
		if grid.LifeGridAt(ntt.Coord()) {
			ntt.ChangeHealth(-2)
			ntt.AddMessage("life cell damage -2")
		}
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
	Entities    map[EntityID]Entity
	GridCoord   GridCoord
	Grid        map[Coord]EntityID
	lifeActive  bool
	lifeAllowed bool
	lifeGrid    [][]bool
	lifePhase   int
	parent      *WorldGrid
	ParentQueue chan DeferredMove
	PlayerCount int
	rng         *rand.Rand
	size        GridSize
}

func NewSubGrid(gcoord GridCoord, sizer Sizer) *SubGrid {
	dgc := NewDunGenCache(10, DungeonEntropy, DungeonProto)
	dgc.InitAtGrid(gcoord)

	size := sizer.GridSize()

	lg := make([][]bool, 2)
	for i := 0; i < 2; i++ {
		lg[i] = make([]bool, size.x*size.y)
	}

	return &SubGrid{
		chatQueue:   make(chan string, (size.x * size.y)),
		deaths:      make([]EntityID, 0, 10),
		dunGenCache: dgc,
		Entities:    make(map[EntityID]Entity),
		GridCoord:   gcoord,
		Grid:        make(map[Coord]EntityID),
		lifeAllowed: true,
		lifeGrid:    lg,
		ParentQueue: make(chan DeferredMove, ((2 * size.x) + (2 * size.y))),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		size:        size,
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
func (self *SubGrid) GridSize() GridSize {
	return self.size
}
func (self *SubGrid) LifeActive() bool {
	return self.lifeActive
}
func (self *SubGrid) LifeGridAt(loc Coord) bool {
	local := loc.LCoord(self)
	return self.lifeGrid[self.lifePhase][local.y*self.size.x+local.x]
}
func (self *SubGrid) MarkDead(ntt Entity) {
	self.deaths = append(self.deaths, ntt.EntityID())
}

const subgrid_placement_trys = 100

func (self *SubGrid) MoveEntity(ntt Entity, loc Coord) {
	if ntt.Coord() != loc {
		if loc.Grid(self) != self.GridCoord {
			ERROR.Panic(`Should not be here!`)
		} else {
			delete(self.Grid, ntt.Coord())
			self.Grid[loc] = ntt.EntityID()
			ntt.SetCoord(loc)
		}
	}
}
func (self *SubGrid) RandomCoord() Coord {
	lx := int64(self.rng.Intn(self.size.x))
	ly := int64(self.rng.Intn(self.size.y))
	x := (self.GridCoord.x * int64(self.size.x)) + lx
	y := (self.GridCoord.y * int64(self.size.y)) + ly
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
	return (coord.Grid(self) != self.GridCoord)
}
func (self *SubGrid) PassableAt(loc Coord) bool {
	return self.EmptyAt(loc) && self.WalkableAt(loc)
}
func (self *SubGrid) PutEntityAt(ntt Entity, loc Coord) {
	if loc.Grid(self) != self.GridCoord {
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
func (self *SubGrid) SetLifeGridAt(loc Coord, value bool) {
	local := loc.LCoord(self)
	self.lifeGrid[self.lifePhase][local.y*self.size.x+local.x] = value
}
func (self *SubGrid) SwapEntities(ntt, other Entity) {
	nttLoc, otherLoc := ntt.Coord(), other.Coord()
	grid1, grid2 := nttLoc.Grid(self), otherLoc.Grid(self)
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
			if player.InMaxRange(ntt) {
				ntt.WriteFor(player, buffer)
				ntt.Detect(player)
			}
		}
	}
}

/*Any live cell with fewer than two live neighbours dies, as if caused by under-population.
Any live cell with two or three live neighbours lives on to the next generation.
Any live cell with more than three live neighbours dies, as if by overcrowding.
Any dead cell with exactly three live neighbours becomes a live cell,
	as if by reproduction.*/
func (self *SubGrid) lifeGridLocal(x, y int) bool {
	if x >= 0 && x < self.size.x && y >= 0 && y < self.size.y {
		return self.lifeGrid[self.lifePhase][y*self.size.x+x]
	}
	return false
}
func (self *SubGrid) lifeGridNeighbors(x, y int) int {
	n := 0
	for j := y - 1; j <= y+1; j++ {
		for i := x - 1; i <= x+1; i++ {
			if (x != i || y != j) && self.lifeGridLocal(i, j) {
				n++
			}
		}
	}
	return n
}
func (self *SubGrid) updateLifeGrid() {
	var nextPhase = (self.lifePhase + 1) % 2
	for y := 0; y < self.size.y; y++ {
		for x := 0; x < self.size.x; x++ {
			n := self.lifeGridNeighbors(x, y)
			if self.lifeGridLocal(x, y) {
				self.lifeGrid[nextPhase][y*self.size.x+x] = (n == 2 || n == 3)
			} else {
				self.lifeGrid[nextPhase][y*self.size.x+x] = (n == 3)
			}
		}
	}
	self.lifePhase = nextPhase
}
func (self *SubGrid) UpdateMovers(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		if ntt.FlagAt(LifeActivateTogl) {
			ntt.ClearFlag(LifeActivateTogl)
			if self.lifeAllowed {
				self.lifeActive = !self.lifeActive
				if self.lifeActive {
					ntt.AddMessage("Life System Activated")
				} else {
					ntt.AddMessage("Life System Dectivated")
				}
			} else {
				self.lifeActive = false
			}
		}
		if ntt.FlagAt(LifeCellTogl) {
			ntt.ClearFlag(LifeCellTogl)
			if self.lifeAllowed {
				loc := ntt.Coord()
				value := self.LifeGridAt(loc)
				self.SetLifeGridAt(loc, !value)
			}
		}
		if ntt.HasMove(gproc) {
			loc := ntt.CalcMove(self)
			ExecuteMove(ntt, self, loc)
		}
		if ntt.IsDead() {
			self.MarkDead(ntt)
		}
	}
	if self.lifeActive { //&& (gproc.TickNumber()%4 == 0) {
		self.updateLifeGrid()
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

	// direction
	buffer.WriteString(`"d":"`)
	buffer.WriteRune(ntt.Direction())
	buffer.WriteString(`",`)

	buffer.WriteString(`"health":`)
	buffer.WriteString(strconv.FormatInt(int64(ntt.Health()), 10))
	buffer.WriteRune(',')

	if gproc.TickNumber()%7 == 0 {
		self.dunGenCache.WriteBasicMap(ntt, buffer)
	} else {
		self.dunGenCache.WriteMap(ntt, buffer)
	}
	buffer.WriteRune(',')

	buffer.WriteString(`"entities":`)
	// This call to parent works concurrently. It's read only.
	// It has to be the parent to coordinate all visible SubGrids
	self.parent.WriteEntities(ntt, buffer)
	buffer.WriteRune(',')

	buffer.WriteString(`"li":`)
	self.parent.WriteLife(ntt, buffer)
	buffer.WriteRune(',')

	if self.lifeAllowed {
		buffer.WriteString(`"la":1,`)
	}

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
	size        GridSize
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
		size:        GridSize{subgrid_width, subgrid_height},
		spawnGrids:  spawnGrids,
	}
}

func (self *WorldGrid) subgridAtGrid(gridCoord GridCoord) *SubGrid {
	subgrid, present := self.grid[gridCoord]
	if !present {
		subgrid = NewSubGrid(gridCoord, self)
		subgrid.parent = self
		self.grid[gridCoord] = subgrid
	}
	return subgrid
}
func (self *WorldGrid) safeSubgridAtGrid(gridCoord GridCoord) *SubGrid {
	subgrid, present := self.grid[gridCoord]
	if !present {
		return nil
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
		if subgrid.Count() == 0 && !subgrid.lifeActive {
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
			// monster population
			for tries := 0; ok && tries < 3; tries++ {
				monster := NewMonster(NewEntityID(), self)
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
	visibleGrids := coord.VisibleGrids(39, 12, self, gcoords[:])
	buffer.WriteRune('"')
	for _, gcoord := range visibleGrids {
		subgrid, present := self.grid[gcoord]
		if present {
			subgrid.WriteEntities(player, buffer)
		}
	}
	buffer.WriteRune('"')
}

func (self *WorldGrid) WriteLife(player Entity, buffer *bytes.Buffer) {
	coord := player.Coord()
	corner := Coord{coord.x - int64(self.size.x/2), coord.y - int64(self.size.y/2)}
	buffer.WriteRune('"')
	var x, y int64
	i, v := 0, 0
	for y = 0; int(y) < self.size.y; y++ {
		for x = 0; int(x) < self.size.x; x++ {
			if self.LifeGridAt(Coord{corner.x + x, corner.y + y}) {
				v += int(1 << uint32(i%6))
			}
			if (i+1)%6 == 0 {
				buffer.WriteRune(Base64Runes[v])
				v = 0
			}
			i += 1
		}
	}
	if i%6 != 0 {
		buffer.WriteRune(Base64Runes[v])
	}
	buffer.WriteRune('"')
}

func (self *WorldGrid) DungeonAt(coord Coord) int8 {
	return self.dunGenCache.DungeonAt(coord)
}

func (srv *WorldGrid) WalkableAt(coord Coord) bool {
	return srv.dunGenCache.WalkableAt(coord)
}

func (self *WorldGrid) DeferMove(ntt Entity, loc Coord) {}
func (self *WorldGrid) EmptyAt(loc Coord) bool {
	subgrid, present := self.grid[loc.Grid(self)]
	if !present {
		return true
	}
	return subgrid.EmptyAt(loc)
}
func (self *WorldGrid) EntityAt(loc Coord) (Entity, bool) {
	gridCoord := loc.Grid(self)
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
func (self *WorldGrid) GridSize() GridSize {
	return self.size
}
func (self *WorldGrid) LifeActive() bool { return false }
func (self *WorldGrid) LifeGridAt(loc Coord) bool {
	subgrid := self.subgridAtGrid(loc.Grid(self))
	return subgrid.LifeGridAt(loc)
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
	gc2 := loc.Grid(self)
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
		i := rand.Intn(len(self.spawnGrids))
		gridCoord := self.spawnGrids[i]
		subgrid := self.subgridAtGrid(gridCoord)
		newEntity, ok = subgrid.NewEntity(ntt)
		if ok {
			self.entityGrid[ntt.EntityID()] = gridCoord
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
	gridCoord := loc.Grid(self)
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
func (self *WorldGrid) SetLifeGridAt(loc Coord, value bool) {
	subgrid := self.subgridAtGrid(loc.Grid(self))
	subgrid.SetLifeGridAt(loc, value)
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
	var wg sync.WaitGroup
	wg.Add(n)
	for _, subgrid := range self.grid {
		go func(sg *SubGrid) {
			defer wg.Done()
			doWork(sg)
		}(subgrid)
	}
	wg.Wait()
}
func (self *WorldGrid) PlayerExec(doWork func(Entity, GridKeeper, GridProcessor), gproc GridProcessor) {
	n := self.playerCount()
	var wg sync.WaitGroup
	wg.Add(n)
	for _, subgrid := range self.grid {
		for _, ntt := range subgrid.Entities {
			if ntt.IsPlayer() {
				go func(e Entity, sg GridKeeper, gp GridProcessor) {
					defer wg.Done()
					doWork(e, sg, gp)
				}(ntt, subgrid, gproc)
			}
		}
	}
	wg.Wait()
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
	self.PlayerExec(func(ntt Entity, grid GridKeeper, gproc GridProcessor) {
		ntt.SendDisplay(grid, gproc)
	}, gproc)
}
func (self *WorldGrid) WriteDisplay(ntt Entity, gproc GridProcessor, buffer *bytes.Buffer) {
	ERROR.Panic("Should not call on World!")
}
