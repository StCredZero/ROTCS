package main

import (
	"bytes"
	"github.com/golang/groupcache/lru"
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

func (self *DunGenCache) WriteBasicMap(ntt Creature, buffer *bytes.Buffer) {
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
}
