package main

import (
	"math/rand"
	"strconv"
)

const subgrid_width = 79
const subgrid_height = 25

type Coord struct {
	x int64
	y int64
}

func loc2grid(d int64, dimSize int64) int64 {
	if d >= 0 {
		return (d / dimSize)
	} else {
		return ((d - (dimSize - 1)) / dimSize)
	}
}

func (self *Coord) Grid() Coord {
	return Coord{
		x: loc2grid(self.x, subgrid_width),
		y: loc2grid(self.y, subgrid_height),
	}
}

func randomSubgridCoord() Coord {
	return Coord{
		x: int64(rand.Intn(subgrid_width)),
		y: int64(rand.Intn(subgrid_height)),
	}
}

func (self *Coord) VisibleGrids(xdist int64, ydist int64) []Coord {
	set := make(map[Coord]bool)
	var c1 = Coord{self.x - xdist, self.y - ydist}
	var c2 = Coord{self.x - xdist, self.y + ydist}
	var c3 = Coord{self.x + xdist, self.y - ydist}
	var c4 = Coord{self.x + xdist, self.y + ydist}
	set[c1.Grid()] = true
	set[c2.Grid()] = true
	set[c3.Grid()] = true
	set[c4.Grid()] = true
	grids := make([]Coord, 4, 4)
	var i int = 0
	for coord, _ := range set {
		grids[i] = coord
		i++
	}
	return grids[:len(set)]
}

func (self *Coord) IndexString() string {
	return `"` + strconv.FormatInt(self.x, 10) + "," + strconv.FormatInt(self.y, 10) + `"`
}