package main

import (
	"bytes"
	"math/rand"
)

type GridProcessor interface {
	DungeonAt(Coord) int
	WalkableAt(Coord) bool
	WriteDisplay(Creature, *bytes.Buffer)
}

type GridKeeper interface {
	EmptyAt(Coord) bool
	MoveEntity(Creature, Coord)
	NewEntity(Creature) (Creature, bool)
	PutEntityAt(Creature, Coord)
	RemoveEntityID(EntityID)
	UpdateMovers(GridProcessor)
	SendDisplays(GridProcessor)
	WriteEntities(Creature, *bytes.Buffer)
}

type SubGrid struct {
	GridCoord   GridCoord
	Grid        map[Coord]EntityID
	Entities    map[EntityID]Creature
	ParentQueue chan EntityID
}

func NewSubGrid(gcoord GridCoord) *SubGrid {
	return &SubGrid{
		GridCoord:   gcoord,
		Grid:        make(map[Coord]EntityID),
		Entities:    make(map[EntityID]Creature),
		ParentQueue: make(chan EntityID, (subgrid_width * subgrid_height)),
	}
}

func (self *SubGrid) EmptyAt(loc Coord) bool {
	_, present := self.Grid[loc]
	return !present
}

const subgrid_placement_trys = 100

func (self *SubGrid) MoveEntity(ntt Creature, loc Coord) {
	if ntt.Coord() != loc {
		if loc.Grid() != self.GridCoord {
			self.ParentQueue <- ntt.EntityID()
		} else {
			delete(self.Grid, ntt.Coord())
			self.Grid[loc] = ntt.EntityID()
			ntt.SetCoord(loc)
			ntt.MoveCommit()
		}
	}
}

func (self *SubGrid) NewEntity(ntt Creature) (Creature, bool) {
	var loc = randomSubgridCoord()
	for n := 0; (!self.EmptyAt(loc)) && (n < subgrid_placement_trys); n++ {
		loc = randomSubgridCoord()
	}
	if !self.EmptyAt(loc) {
		return &Entity{}, false
	}
	ntt.SetCoord(loc)
	self.Entities[ntt.EntityID()] = ntt
	self.Grid[loc] = ntt.EntityID()
	return ntt, true
}

func (self *SubGrid) RemoveEntityID(id EntityID) {
	ntt, present := self.Entities[id]
	if !present {
		panic("Removing nonexistent Entity")
	}
	delete(self.Grid, ntt.Coord())
	delete(self.Entities, id)
}

func (self *SubGrid) PutEntityAt(ntt Creature, loc Coord) {
	ntt.SetCoord(loc)
	self.Grid[loc] = ntt.EntityID()
	self.Entities[ntt.EntityID()] = ntt
}

func (self *SubGrid) WriteEntities(player Creature, buffer *bytes.Buffer) {
	for _, id := range self.Grid {
		if id != player.EntityID() {
			ntt := self.Entities[id]
			ntt.WriteFor(player, buffer)
			buffer.WriteString(`,`)
		}
	}
}

func (self *SubGrid) UpdateMovers(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		ntt.Move(self, gproc)
	}
}

func (self *SubGrid) SendDisplays(gproc GridProcessor) {
	for _, ntt := range self.Entities {
		ntt.SendDisplay(self, gproc)
	}
}

type WorldGrid struct {
	grid       map[GridCoord]*SubGrid
	entityGrid map[EntityID]GridCoord
	spawnGrids []GridCoord
}

func NewWorldGrid() *WorldGrid {
	spawnGrids := make([]GridCoord, 1)
	spawnGrids[0] = GridCoord{0, 0}
	return &WorldGrid{
		grid:       make(map[GridCoord]*SubGrid),
		entityGrid: make(map[EntityID]GridCoord),
		spawnGrids: spawnGrids,
	}
}

func (self *WorldGrid) subgridAtGrid(gridCoord GridCoord) *SubGrid {
	subgrid, present := self.grid[gridCoord]
	if !present {
		subgrid = NewSubGrid(gridCoord)
		self.grid[gridCoord] = subgrid
	}
	return subgrid
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
		panic("Moving nonexistent Entity")
	}
	sg1 := self.subgridAtGrid(gc1)
	gc2 := loc.Grid()
	if gc1 == gc2 {
		println("non-global move")
		sg1.MoveEntity(ntt, loc)
	} else {
		println("global move")
		sg2 := self.subgridAtGrid(gc2)
		sg1.RemoveEntityID(ntt.EntityID())
		sg2.PutEntityAt(ntt, loc)
		ntt.MoveCommit()
		self.entityGrid[ntt.EntityID()] = gc2
	}
}

func (self *WorldGrid) NewEntity(ntt Creature) (Creature, bool) {
	var newEntity Creature
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

func (self *WorldGrid) PutEntityAt(ntt Creature, loc Coord) {
	_, present := self.entityGrid[ntt.EntityID()]
	if present {
		panic("Placing already existing Entity")
	}
	gridCoord := loc.Grid()
	self.entityGrid[ntt.EntityID()] = gridCoord
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.PutEntityAt(ntt, loc)
}

func (self *WorldGrid) RemoveEntityID(id EntityID) {
	gridCoord, present := self.entityGrid[id]
	if !present {
		panic("Removing nonexistent Entity")
	}
	delete(self.entityGrid, id)
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.RemoveEntityID(id)
}

func (self *WorldGrid) UpdateMovers(gproc GridProcessor) {
	for _, subgrid := range self.grid {
		subgrid.UpdateMovers(gproc)
	}
	for _, subgrid := range self.grid {
		done := false
		for !done {
			select {
			case id := <-subgrid.ParentQueue:
				ntt := subgrid.Entities[id]
				ntt.Move(self, gproc)
			default:
				done = true
			}
		}
	}
}

func (self *WorldGrid) SendDisplays(gproc GridProcessor) {
	for _, subgrid := range self.grid {
		subgrid.SendDisplays(gproc)
	}
}
