package main

import (
	"bytes"
	"github.com/golang/groupcache/lru"
	"strconv"
)

type DunGenCache struct {
	entropy DunGenEntropy
	cache   *lru.Cache
	proto   DunGen
}

func NewDunGenCache(maxEntries int, entropy DunGenEntropy, proto DunGen) *DunGenCache {
	return &DunGenCache{
		entropy: entropy,
		cache:   lru.New(maxEntries),
		proto:   proto,
	}
}

func (self *DunGenCache) basicDungeonAt(gcoord GridCoord) *DunGen {
	var intf interface{}
	var present bool
	intf, present = self.cache.Get(lru.Key(gcoord))
	if present {
		var d *DunGen = intf.(*DunGen)
		return d
	} else {
		newdg := NewDunGen(&self.proto)
		newdg.createDungeon(gcoord, self.entropy)
		self.cache.Add(lru.Key(gcoord), newdg)
		return newdg
	}
}

func (self *DunGenCache) DungeonAtGrid(gcoord GridCoord) *DunGen {
	dg := self.basicDungeonAt(gcoord)
	if !dg.passaged {
		dgn := self.basicDungeonAt(GridCoord{gcoord.x, gcoord.y - 1})
		dgw := self.basicDungeonAt(GridCoord{gcoord.x - 1, gcoord.y})
		dg.makePassages(dgn, dgw)
	}
	return dg
}

func (self *DunGenCache) DungeonAt(coord Coord) int8 {
	dgrid := self.DungeonAtGrid(coord.Grid())
	lcoord := coord.LCoord()
	return dgrid.TileAt(lcoord)
}

func (self *DunGenCache) WalkableAt(coord Coord) bool {
	dgrid := self.DungeonAtGrid(coord.Grid())
	lcoord := coord.LCoord()
	return dgrid.isWalkable(lcoord.x, lcoord.y)
}

func (self *DunGenCache) WriteMap(ntt Creature, buffer *bytes.Buffer) {
	if ntt.Initialized() && manhattanDist(ntt.Coord(), ntt.LastDispCoord()) == 0 {
		self.WriteEntityMap(ntt, buffer)
	} else if ntt.Initialized() && manhattanDist(ntt.Coord(), ntt.LastDispCoord()) == 1 {
		self.WriteLineMap(ntt, buffer)
	} else {
		self.WriteBasicMap(ntt, buffer)
	}
}

func (self *DunGenCache) WriteEntityMap(ntt Creature, buffer *bytes.Buffer) {
	buffer.WriteString(`"maptype":"entity"`)
}

func (self *DunGenCache) WriteLineMap(ntt Creature, buffer *bytes.Buffer) {
	corner := ntt.Coord().Corner()
	move := ntt.LastDispCoord().AsMoveTo(ntt.Coord())
	var start Coord
	switch move {
	case 's':
		start = Coord{corner.x, corner.y + subgrid_height - 1}
	case 'e':
		start = Coord{corner.x + subgrid_width - 1, corner.y}
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
		for x := start.x; x < (start.x + subgrid_width); x++ {
			cell := self.DungeonAt(Coord{x, start.y})
			switch cell {
			case TileFloor, TileCorridor:
				buffer.WriteRune('.')
			default:
				buffer.WriteRune(' ')
			}
		}
	case 'w', 'e':
		for y := start.y; y < (start.y + subgrid_height); y++ {
			cell := self.DungeonAt(Coord{start.x, y})
			switch cell {
			case TileFloor, TileCorridor:
				buffer.WriteRune('.')
			default:
				buffer.WriteRune(' ')
			}
		}
	}
	buffer.WriteRune('"')
}

func (self *DunGenCache) WriteBasicMap(ntt Creature, buffer *bytes.Buffer) {
	buffer.WriteString(`"maptype":"basic",`)
	buffer.WriteString(`"map":`)

	var x, y, xstart, ystart, xend, yend int64
	xstart = ntt.Coord().x - (subgrid_width / 2)
	ystart = ntt.Coord().y - (subgrid_height / 2)
	xend, yend = xstart+subgrid_width, ystart+subgrid_height
	buffer.WriteRune('[')
	for y = ystart; y < yend; y++ {
		buffer.WriteRune('"')
		for x = xstart; x < xend; x++ {
			cell := self.DungeonAt(Coord{x, y})
			switch cell {
			case TileFloor, TileCorridor:
				buffer.WriteRune('.')
			default:
				buffer.WriteRune(' ')
			}
		}
		buffer.WriteString(`",`)
	}
	buffer.WriteString(`"e"]`)
	ntt.SetInitialized(true)
}
