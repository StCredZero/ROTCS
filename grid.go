package main

type SubGrid struct {
	GridCoord Coord
	Grid      map[Coord]uint32
	Entities  map[uint32]Entity
}

func (self *SubGrid) EmptyAt(loc Coord) bool {
	_, present := self.Grid[loc]
	return !present
}

func (self *SubGrid) NewEntity(id uint32, trys int) (bool, Entity) {
	var loc = randomSubgridCoord()
	for n := 0; (!self.EmptyAt(loc)) && (n < trys); n++ {
		loc = randomSubgridCoord()
	}
	if self.EmptyAt(loc) {
		return false, Entity{}
	}
	self.Entities[id] = Entity{
		Id:       id,
		Location: loc,
	}
	self.Grid[loc] = id
	return true, self.Entities[id]
}

func (self *SubGrid) PutEntityAt(ntt Entity, loc Coord) {
	ntt.Location = loc
	self.Grid[loc] = ntt.Id
	self.Entities[ntt.Id] = ntt
}

type WorldGrid struct {
	grid       map[Coord]SubGrid
	entityGrid map[uint32]Coord
}

/*func (self *WorldGrid) NewEntity(id uint32) Entity {

}*/
