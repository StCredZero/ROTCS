package main

import (
	"bytes"
	"math/rand"
)

type GridUpdateFn func(GridKeeper, Entity, GridProcessor)

type GridProcessor interface {
	ProcessEntities(GridUpdateFn, *CstServer)
	DisplayFor(Entity) string
}

type GridKeeper interface {
	DisplayFor(Entity) string
	EmptyAt(Coord) bool
	MoveEntity(Entity, Coord)
	NewEntity(Entity) (Entity, bool)
	PutEntityAt(Entity, Coord)
	RemoveEntityId(EntityId)
	RewriteEntity(Entity)
	UpdateEntities(GridUpdateFn, GridProcessor)
}

type SubGrid struct {
	GridCoord   GridCoord
	Grid        map[Coord]EntityId
	Entities    map[EntityId]Entity
	ParentQueue chan EntityId
}

func NewSubGrid(gcoord GridCoord) *SubGrid {
	return &SubGrid{
		GridCoord:   gcoord,
		Grid:        make(map[Coord]EntityId),
		Entities:    make(map[EntityId]Entity),
		ParentQueue: make(chan EntityId, (subgrid_width * subgrid_height)),
	}
}

func (self *SubGrid) EmptyAt(loc Coord) bool {
	_, present := self.Grid[loc]
	return !present
}

const subgrid_placement_trys = 100

func (self *SubGrid) MoveEntity(ntt Entity, loc Coord) {
	if ntt.Location != loc {
		if loc.Grid() != self.GridCoord {
			self.ParentQueue <- ntt.Id
		} else {
			delete(self.Grid, ntt.Location)
			self.Grid[loc] = ntt.Id
			ntt.Location = loc
			self.Entities[ntt.Id] = ntt
		}
	}
}

func (self *SubGrid) NewEntity(ntt Entity) (Entity, bool) {
	var loc = randomSubgridCoord()
	for n := 0; (!self.EmptyAt(loc)) && (n < subgrid_placement_trys); n++ {
		loc = randomSubgridCoord()
	}
	if !self.EmptyAt(loc) {
		return Entity{}, false
	}
	ntt.Location = loc
	self.Entities[ntt.Id] = ntt
	self.Grid[loc] = ntt.Id
	return ntt, true
}

func (self *SubGrid) RemoveEntityId(id EntityId) {
	ntt, present := self.Entities[id]
	if !present {
		panic("Removing nonexistent Entity")
	}
	delete(self.Grid, ntt.Location)
	delete(self.Entities, id)
}

func (self *SubGrid) PutEntityAt(ntt Entity, loc Coord) {
	ntt.Location = loc
	self.Grid[loc] = ntt.Id
	self.Entities[ntt.Id] = ntt
}

func (self *SubGrid) RewriteEntity(ntt Entity) {
	oldEntity := self.Entities[ntt.Id]
	ntt.Location = oldEntity.Location
	self.Entities[ntt.Id] = ntt
}

func (self *SubGrid) Corner() Coord {
	return Coord{
		x: int64(self.GridCoord.x * subgrid_width),
		y: int64(self.GridCoord.y * subgrid_height),
	}
}

func (self *SubGrid) DisplayFor(player Entity) string {
	var first bool = true
	var buffer bytes.Buffer
	buffer.WriteString(`{"type":"update","data":{"maptype":"entity","entities":{`)
	for _, id := range self.Grid {
		if !first {
			buffer.WriteString(",")
		}
		ntt := self.Entities[id]
		buffer.WriteString(ntt.DisplayString())
		first = false
	}
	buffer.WriteString(`}}}`)
	return buffer.String()
}

func (self *SubGrid) UpdateEntities(updateFn GridUpdateFn, gproc GridProcessor) {
	for _, ntt := range self.Entities {
		updateFn(self, ntt, gproc)
	}
}

type WorldGrid struct {
	grid       map[GridCoord]*SubGrid
	entityGrid map[EntityId]GridCoord
	spawnGrids []GridCoord
}

func NewWorldGrid() *WorldGrid {
	spawnGrids := make([]GridCoord, 1)
	spawnGrids[0] = GridCoord{0, 0}
	return &WorldGrid{
		grid:       make(map[GridCoord]*SubGrid),
		entityGrid: make(map[EntityId]GridCoord),
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

func (self *WorldGrid) DisplayFor(ntt Entity) string {
	subgrid := self.subgridAtGrid(ntt.Location.Grid())
	return subgrid.DisplayFor(ntt)
}

func (self *WorldGrid) EmptyAt(loc Coord) bool {
	subgrid, present := self.grid[loc.Grid()]
	if !present {
		return true
	}
	return subgrid.EmptyAt(loc)
}

func (self *WorldGrid) MoveEntity(ntt Entity, loc Coord) {
	gc1, present := self.entityGrid[ntt.Id]
	if !present {
		panic("Moving nonexistent Entity")
	}
	sg1 := self.subgridAtGrid(gc1)
	gc2 := loc.Grid()
	if gc1 == gc2 {
		sg1.MoveEntity(ntt, loc)
	} else {
		sg2 := self.subgridAtGrid(gc2)
		sg1.RemoveEntityId(ntt.Id)
		sg2.PutEntityAt(ntt, loc)
	}
}

func (self *WorldGrid) NewEntity(ntt Entity) (Entity, bool) {
	var newEntity Entity
	ok := false
	for !ok {
		i := rand.Intn(len(self.spawnGrids))
		gridCoord := self.spawnGrids[i]
		subgrid := self.subgridAtGrid(gridCoord)
		newEntity, ok = subgrid.NewEntity(ntt)
	}
	return newEntity, ok
}

func (self *WorldGrid) PutEntityAt(ntt Entity, loc Coord) {
	_, present := self.entityGrid[ntt.Id]
	if present {
		panic("Placing already existing Entity")
	}
	gridCoord := loc.Grid()
	self.entityGrid[ntt.Id] = gridCoord
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.PutEntityAt(ntt, loc)
}

func (self *WorldGrid) RemoveEntityId(id EntityId) {
	gridCoord, present := self.entityGrid[id]
	if !present {
		panic("Removing nonexistent Entity")
	}
	delete(self.entityGrid, id)
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.RemoveEntityId(id)
}

func (self *WorldGrid) RewriteEntity(ntt Entity) {
	gridCoord, present := self.entityGrid[ntt.Id]
	if !present {
		panic("Updating nonexistent Entity")
	}
	subgrid := self.subgridAtGrid(gridCoord)
	subgrid.RewriteEntity(ntt)
}

func (self *WorldGrid) UpdateEntities(updateFn GridUpdateFn, gproc GridProcessor) {
	for _, subgrid := range self.grid {
		subgrid.UpdateEntities(updateFn, gproc)
	}
}
