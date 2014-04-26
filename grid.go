package main

import (
	"bytes"
	"math/rand"
)

type GridKeeper interface {
	DisplayString() string
	EmptyAt(Coord) bool
	MoveEntity(Entity, Coord)
	NewEntity(Entity) (Entity, bool)
	PutEntityAt(Entity, Coord)
	RewriteEntity(Entity)
	UpdateEntities(func(GridKeeper, Entity), *CstServer)
}

type UpdateFunc func(*GridKeeper, Entity)

type SubGrid struct {
	GridCoord   Coord
	Grid        map[Coord]uint32
	Entities    map[uint32]Entity
	ParentQueue chan uint32
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

func (self *SubGrid) DisplayString() string {
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

func (self *SubGrid) UpdateEntities(updateFn func(GridKeeper, Entity), server *CstServer) {
	for _, ntt := range self.Entities {
		updateFn(self, ntt)
	}
}

type WorldGrid struct {
	grid       map[Coord]SubGrid
	entityGrid map[uint32]Coord
	spawnGrids []Coord
}

func (self *WorldGrid) SubGridAtGrid(gridCoord Coord) SubGrid {
	subgrid, present := self.grid[gridCoord]
	if !present {
		subgrid = SubGrid{
			GridCoord:   gridCoord,
			Grid:        make(map[Coord]uint32),
			Entities:    make(map[uint32]Entity),
			ParentQueue: make(chan uint32, (subgrid_width * subgrid_height)),
		}
		self.grid[gridCoord] = subgrid
	}
	return subgrid
}

func (self *WorldGrid) SubGridAt(coord Coord) SubGrid {
	gridCoord := coord.Grid()
	return self.SubGridAtGrid(gridCoord)
}

func (self *WorldGrid) NewEntity(ntt Entity) (Entity, bool) {
	var newEntity Entity
	ok := false
	for !ok {
		i := rand.Intn(len(self.spawnGrids))
		gridCoord := self.spawnGrids[i]
		subgrid := self.SubGridAtGrid(gridCoord)
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
	subgrid := self.SubGridAtGrid(gridCoord)
	subgrid.PutEntityAt(ntt, loc)
}

func (self *WorldGrid) RewriteEntity(ntt Entity) {
	gridCoord, present := self.entityGrid[ntt.Id]
	if !present {
		panic("Updating nonexistent Entity")
	}
	subgrid := self.SubGridAtGrid(gridCoord)
	subgrid.RewriteEntity(ntt)
}

func (self *WorldGrid) UpdateEntities(updateFn func(GridKeeper, Entity), server *CstServer) {
	/*for _, subgrid := range self.grid {
		subgrid.UpdateEntities(updateFn, server)
	}*/
}
