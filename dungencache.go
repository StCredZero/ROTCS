package main

import (
	"bytes"
	"strconv"

	//"github.com/golang/groupcache/lru"
)

type DunGenCache struct {
	entropy DunGenEntropy
	cache   map[GridCoord]*DunGen
	proto   DunGen
}

func NewDunGenCache(maxEntries int, entropy DunGenEntropy, proto DunGen) *DunGenCache {
	dgc := DunGenCache{
		entropy: entropy,
		cache:   make(map[GridCoord]*DunGen),
		proto:   proto,
	}
	return &dgc
}

func (self *DunGenCache) GridSize() GridSize {
	return self.proto.GridSize()
}

func (self *DunGenCache) basicDungeonAt(gcoord GridCoord) *DunGen {
	dg, present := self.cache[gcoord]
	if present {
		return dg
	} else {
		newdg := NewDunGen(&self.proto)
		newdg.createDungeon(gcoord, self.entropy)
		if gcoord == (GridCoord{0, 0}) {
			newdg.readFile("resources/shipmap0")
		}
		self.cache[gcoord] = newdg
		return newdg
	}
}

func (self *DunGenCache) InitAtGrid(gcoord GridCoord) {
	self.DungeonAtGrid(gcoord)
	var x, y int64
	for y = -1; y <= 1; y++ {
		for x = -1; x <= 1; x++ {
			if x != 0 && y != 0 {
				self.DungeonAtGrid(GridCoord{gcoord.x + x, gcoord.y + y})
			}
		}
	}
}

func (self *DunGenCache) DungeonAtGrid(gcoord GridCoord) *DunGen {
	dg := self.basicDungeonAt(gcoord)
	if !dg.passagedNorth {
		dgn := self.basicDungeonAt(GridCoord{gcoord.x, gcoord.y - 1})
		dg.makePassagesNorth(dgn)
	}
	if !dg.passagedSouth {
		dgs := self.basicDungeonAt(GridCoord{gcoord.x, gcoord.y + 1})
		dg.makePassagesSouth(dgs)
	}
	if !dg.passagedEast {
		dge := self.basicDungeonAt(GridCoord{gcoord.x + 1, gcoord.y})
		dg.makePassagesEast(dge)
	}
	if !dg.passagedWest {
		dgw := self.basicDungeonAt(GridCoord{gcoord.x - 1, gcoord.y})
		dg.makePassagesWest(dgw)
	}
	return dg
}

func (self *DunGenCache) DungeonAt(coord Coord) int8 {
	dgrid := self.DungeonAtGrid(coord.Grid(self))
	lcoord := coord.LCoord(self)
	return dgrid.TileAt(lcoord)
}

func (self *DunGenCache) WalkableAt(coord Coord) bool {
	dgrid := self.DungeonAtGrid(coord.Grid(self))
	lcoord := coord.LCoord(self)
	return dgrid.isWalkable(lcoord.x, lcoord.y)
}

func (self *DunGenCache) WriteMap(ntt Entity, buffer *bytes.Buffer) {
	if ntt.Initialized() && manhattanDist(ntt.Coord(), ntt.LastDispCoord()) == 0 {
		self.WriteEntityMap(ntt, buffer)
	} else if ntt.Initialized() && manhattanDist(ntt.Coord(), ntt.LastDispCoord()) == 1 {
		self.WriteLineMap(ntt, buffer)
	} else {
		self.WriteBasicMap(ntt, buffer)
	}
}

func (self *DunGenCache) WriteEntityMap(ntt Entity, buffer *bytes.Buffer) {
	buffer.WriteString(`"maptype":"entity"`)
}

func (self *DunGenCache) WriteLineMap(ntt Entity, buffer *bytes.Buffer) {
	size := self.GridSize()
	corner := ntt.Coord().Corner(self)
	move := ntt.LastDispCoord().AsMoveTo(ntt.Coord())
	var start Coord
	switch move {
	case 's':
		start = Coord{corner.x, corner.y + int64(size.y) - 1}
	case 'e':
		start = Coord{corner.x + int64(size.x) - 1, corner.y}
	default:
		start = corner
	}
	buffer.WriteString(`"maptype":"line",`)
	buffer.WriteString(`"start":[`)
	buffer.WriteString(strconv.FormatInt(start.x, 10))
	buffer.WriteRune(',')
	buffer.WriteString(strconv.FormatInt(start.y, 10))
	buffer.WriteString(`],`)
	buffer.WriteString(`"orientation":"`)
	buffer.WriteRune(move)
	buffer.WriteString(`",`)
	buffer.WriteString(`"line":"`)
	switch move {
	case 'n', 's':
		WriteBase64Map(start, size.x, 1, self, buffer)
	case 'w', 'e':
		WriteBase64Map(start, 1, size.y, self, buffer)
	}
	buffer.WriteRune('"')
}

func (self *DunGenCache) WriteBasicMap(ntt Entity, buffer *bytes.Buffer) {
	size := self.GridSize()
	buffer.WriteString(`"maptype":"basic",`)
	buffer.WriteString(`"map":"`)
	xstart := ntt.Coord().x - (int64(size.x) / 2)
	ystart := ntt.Coord().y - (int64(size.y) / 2)
	WriteBase64Map(Coord{xstart, ystart}, size.x, size.y, self, buffer)
	buffer.WriteRune('"')
	ntt.SetInitialized(true)
}

var Base64Runes = []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '/'}

func WriteBase64Map(corner Coord, xsize int, ysize int, dungeon *DunGenCache, buffer *bytes.Buffer) {
	var v int = 0
	i := 0
	for y := corner.y; y < corner.y+int64(ysize); y++ {
		for x := corner.x; x < corner.x+int64(xsize); x++ {
			if dungeon.WalkableAt(Coord{x, y}) {
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
}
