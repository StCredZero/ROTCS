package main

import (
	"bytes"
	"math/big"
	//"math/rand"
	"strconv"
)

const subgrid_width = 79
const subgrid_height = 25

type Coord struct {
	x int64
	y int64
}

func (self Coord) Grid() GridCoord {
	return GridCoord{
		x: loc2grid(self.x, subgrid_width),
		y: loc2grid(self.y, subgrid_height),
	}
}

func (self Coord) LCoord() LCoord {
	mygrid := self.Grid()
	return LCoord{
		x: int(self.x - (mygrid.x * subgrid_width)),
		y: int(self.y - (mygrid.y * subgrid_height)),
	}
}

func (self Coord) MovedBy(move rune) Coord {
	var result Coord
	switch move {
	case 'n':
		result = Coord{self.x, self.y - 1}
	case 's':
		result = Coord{self.x, self.y + 1}
	case 'w':
		result = Coord{self.x - 1, self.y}
	case 'e':
		result = Coord{self.x + 1, self.y}
	default:
		result = self
	}
	return result
}

func (self Coord) AsMoveTo(other Coord) rune {
	if self.x == other.x && (self.y-1) == other.y {
		return 'n'
	} else if self.x == other.x && (self.y+1) == other.y {
		return 's'
	} else if (self.x-1) == other.x && self.y == other.y {
		return 'w'
	} else if (self.x+1) == other.x && self.y == other.y {
		return 'e'
	} else {
		return '0'
	}
}

func (self Coord) Corner() Coord {
	return Coord{self.x - 39, self.y - 12}
}

func neighbors4(coord Coord) []Coord {
	return []Coord{
		{coord.x, coord.y - 1},
		{coord.x, coord.y + 1},
		{coord.x - 1, coord.y},
		{coord.x + 1, coord.y},
	}
}

func (self Coord) VisibleGrids(xdist int64, ydist int64) []GridCoord {
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

func (self Coord) WriteDisplay(player Creature, buffer *bytes.Buffer) {
	buffer.WriteString(`"`)
	buffer.WriteString(strconv.FormatInt(self.x, 10))
	buffer.WriteString(`,`)
	buffer.WriteString(strconv.FormatInt(self.y, 10))
	buffer.WriteString(`"`)
}

type GridCoord struct {
	x int64
	y int64
}

func (self GridCoord) Corner() Coord {
	return Coord{self.x * subgrid_width, self.y * subgrid_height}
}

func (self GridCoord) Expansion() []GridCoord {
	result := make([]GridCoord, 0, 9)
	var x, y int64
	for y = -1; y <= 1; y++ {
		for x = -1; x <= 1; x++ {
			result = append(result, GridCoord{self.x + x, self.y + y})
		}
	}
	return result
}

func (gridCoord GridCoord) WriteTo(b *bytes.Buffer) (int, error) {
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

type SortableLCoords []LCoord

func (this SortableLCoords) Len() int {
	return len(this)
}
func (this SortableLCoords) Less(i, j int) bool {
	return (this[i].x < this[j].x) ||
		((this[i].x == this[j].x) && (this[i].y < this[j].y))
}
func (this SortableLCoords) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

type SortableGCoords []GridCoord

func (this SortableGCoords) Len() int {
	return len(this)
}
func (this SortableGCoords) Less(i, j int) bool {
	return (this[i].x < this[j].x) ||
		((this[i].x == this[j].x) && (this[i].y < this[j].y))
}
func (this SortableGCoords) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func loc2grid(d int64, dimSize int64) int64 {
	if d >= 0 {
		return (d / dimSize)
	} else {
		return ((d - (dimSize - 1)) / dimSize)
	}
}
