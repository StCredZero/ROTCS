package main

import (
	"bytes"
	"math/big"
	"math/rand"
	"strconv"
)

const subgrid_width = 79
const subgrid_height = 25

type Coord struct {
	x int64
	y int64
}

func (self *Coord) Grid() GridCoord {
	return GridCoord{
		x: loc2grid(self.x, subgrid_width),
		y: loc2grid(self.y, subgrid_height),
	}
}

func (self *Coord) LCoord() LCoord {
	mygrid := self.Grid()
	return LCoord{
		x: int(self.x - (mygrid.x * subgrid_width)),
		y: int(self.y - (mygrid.y * subgrid_height)),
	}
}

func randomSubgridCoord() Coord {
	return Coord{
		x: int64(rand.Intn(subgrid_width)),
		y: int64(rand.Intn(subgrid_height)),
	}
}

func (self *Coord) VisibleGrids(xdist int64, ydist int64) []GridCoord {
	set := make(map[GridCoord]bool)
	var c1 = Coord{self.x - xdist, self.y - ydist}
	var c2 = Coord{self.x - xdist, self.y + ydist}
	var c3 = Coord{self.x + xdist, self.y - ydist}
	var c4 = Coord{self.x + xdist, self.y + ydist}
	set[c1.Grid()] = true
	set[c2.Grid()] = true
	set[c3.Grid()] = true
	set[c4.Grid()] = true
	grids := make([]GridCoord, 4, 4)
	var i int = 0
	for coord, _ := range set {
		grids[i] = coord
		i++
	}
	return grids[:len(set)]
}

func (self *Coord) WriteDisplay(player Displayer, buffer *bytes.Buffer) {
	x := (self.x - player.Coord().x) + (subgrid_width / 2)
	y := (self.y - player.Coord().y) + (subgrid_height / 2)
	buffer.WriteString(`"`)
	buffer.WriteString(strconv.FormatInt(x, 10))
	buffer.WriteString(`,`)
	buffer.WriteString(strconv.FormatInt(y, 10))
	buffer.WriteString(`"`)
}

type GridCoord struct {
	x int64
	y int64
}

func (self *GridCoord) Corner() Coord {
	return Coord{self.x * subgrid_width, self.y * subgrid_height}
}

func (gridCoord *GridCoord) WriteTo(b *bytes.Buffer) (int, error) {
	xn, xerr := b.Write(big.NewInt(gridCoord.x).Bytes())
	if xerr != nil {
		return xn, xerr
	}
	yn, yerr := b.Write(big.NewInt(gridCoord.y).Bytes())
	return xn + yn, yerr
}

type LCoord struct {
	x int
	y int
}

func (self LCoord) inBounds() bool {
	return self.x >= 0 && self.y >= 0 &&
		self.x < subgrid_width && self.y < subgrid_height
}

func (self LCoord) inShyBounds() bool {
	return self.x >= 1 && self.y >= 1 &&
		self.x < subgrid_width-1 && self.y < subgrid_height-1
}

func loc2grid(d int64, dimSize int64) int64 {
	if d >= 0 {
		return (d / dimSize)
	} else {
		return ((d - (dimSize - 1)) / dimSize)
	}
}
