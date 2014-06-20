package main

import (
	"bytes"
	"math"
	"math/big"
	//"math/rand"
	"sort"
)

const subgrid_width = 79
const subgrid_height = 25

type GridSize struct {
	x int
	y int
}

type Sizer interface {
	GridSize() GridSize
}

type Coord struct {
	x int64
	y int64
}

var NullGCoord = GridCoord{math.MaxInt64, math.MaxInt64}

func (self Coord) Grid(sizer Sizer) GridCoord {
	size := sizer.GridSize()
	return GridCoord{
		x: loc2grid(self.x, int64(size.x)),
		y: loc2grid(self.y, int64(size.y)),
	}
}

func (self Coord) LCoord(sizer Sizer) LCoord {
	size := sizer.GridSize()
	mygrid := self.Grid(sizer)
	return LCoord{
		x: int(self.x - (mygrid.x * int64(size.x))),
		y: int(self.y - (mygrid.y * int64(size.y))),
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

func (self Coord) VisibleGrids(xdist int64, ydist int64, sizer Sizer, gcoords []GridCoord) []GridCoord {
	var c0, c1 = Coord{self.x - xdist, self.y - ydist}, Coord{self.x - xdist, self.y + ydist}
	var c2, c3 = Coord{self.x + xdist, self.y - ydist}, Coord{self.x + xdist, self.y + ydist}
	gcoords[0], gcoords[1], gcoords[2], gcoords[3] = c0.Grid(sizer), c1.Grid(sizer), c2.Grid(sizer), c3.Grid(sizer)
	sort.Sort(SortableGCoords(gcoords))
	var count int = 4
	prev := NullGCoord
	for i := 0; i < 4; i++ {
		if gcoords[i] == prev {
			count--
			gcoords[i] = NullGCoord
		} else {
			prev = gcoords[i]
		}
	}
	sort.Sort(SortableGCoords(gcoords))
	return gcoords[:count]
}

var Base91Table = []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '!', '#', '$', '%', '&', '(', ')', '*', '+', ',', '.', '/', ':', ';', '<', '=', '>', '?', '@', '[', ']', '^', '_', '`', '{', '|', '}', '~', '-'}

func (self Coord) WriteDisplay(player Entity, buffer *bytes.Buffer) {
	dx := self.x - player.Coord().x + (subgrid_width / 2)
	dy := self.y - player.Coord().y + (subgrid_height / 2)
	buffer.WriteRune(Base91Table[dx])
	buffer.WriteRune(Base91Table[dy])
}

func (self Coord) InRange(other Coord, xrange int, yrange int) bool {
	return abs(self.x-other.x) <= int64(xrange) &&
		abs(self.y-other.y) <= int64(yrange)
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
