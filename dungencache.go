package main

import "github.com/golang/groupcache/lru"

type DunGenCache struct {
	entropy []byte
	cache   *lru.Cache
	proto   DunGen
}

func NewDunGenCache(maxEntries int, entropy []byte, proto DunGen) *DunGenCache {
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

func (self *DunGenCache) DungeonAt(gcoord GridCoord) *DunGen {
	dg := self.basicDungeonAt(gcoord)
	dgn := self.basicDungeonAt(GridCoord{gcoord.x, gcoord.y - 1})
	dgw := self.basicDungeonAt(GridCoord{gcoord.x - 1, gcoord.y})
	dg.makePassages(dgn, dgw)
	return dg
}
