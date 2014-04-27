package main

import (
	"bytes"
	"fmt"
	"math/rand"
)

const TILE_UNUSED = 0
const TILE_WALL = 1
const TILE_FLOOR = 2
const TILE_STONEWALL = 3
const TILE_CORRIDOR = 4
const TILE_DOOR = 5

const minRoomDim = 2
const maxRoomWidth = 28
const maxRoomHeight = 14
const minCorridorDim = 2
const maxLengthCorridor = 16
const maxRooms = 100

type TileDelta struct {
	x int
	y int
}

const DIR_NORTH = 0
const DIR_SOUTH = 1
const DIR_EAST = 2
const DIR_WEST = 3

type DRect struct {
	x int
	y int
	w int
	h int
}

func (self *DRect) bottom() int {
	return self.y + self.h
}

func (self *DRect) right() int {
	return self.x + self.w
}

type DunGen struct {
	xsize     int
	ysize     int
	objects   int
	targetObj int

	numRooms   int
	chanceRoom int

	dungeon_map [subgrid_width * subgrid_height]int

	walls [4](map[LCoord]bool)
	rooms []DRect

	rng *rand.Rand
}

func (self *DunGen) initialize() *DunGen {
	self.xsize = subgrid_width
	self.ysize = subgrid_height
	self.targetObj = 20
	self.chanceRoom = 50

	self.rooms = make([]DRect, 100, 100)
	for i := 0; i < 4; i++ {
		self.walls[i] = make(map[LCoord]bool)
	}
	//self.dungeon_map = make([]int, subgrid_width*subgrid_height)
	return self
}

func (self *DunGen) setCell(x int, y int, value int) {
	if x >= 0 && x < self.xsize && y >= 0 && y < self.ysize {
		self.dungeon_map[x+(self.xsize*y)] = value
		self.clearWalls(x, y)
	}
}

func (self *DunGen) getCell(x int, y int) int {
	if x >= 0 && x < self.xsize && y >= 0 && y < self.ysize {
		return self.dungeon_map[x+(self.xsize*y)]
	} else {
		return TILE_UNUSED
	}
}

func (self *DunGen) isWalkable(x int, y int) bool {
	return ((self.getCell(x, y) == TILE_FLOOR) ||
		(self.getCell(x, y) == TILE_CORRIDOR) ||
		(self.getCell(x, y) == TILE_DOOR))
}

func (self *DunGen) debugPrint() string {
	var buffer bytes.Buffer
	for y := 0; y < self.ysize; y++ {
		for x := 0; x < self.xsize; x++ {
			if (self.walls[DIR_NORTH])[LCoord{x, y}] {
				buffer.WriteString("N")
			} else if (self.walls[DIR_SOUTH])[LCoord{x, y}] {
				buffer.WriteString("S")
			} else if (self.walls[DIR_WEST])[LCoord{x, y}] {
				buffer.WriteString("W")
			} else if (self.walls[DIR_EAST])[LCoord{x, y}] {
				buffer.WriteString("E")
			} else if self.isWalkable(x, y) {
				buffer.WriteString(".")
			} else {
				buffer.WriteString("0")
			}
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (self *DunGen) getRand(min int, max int) int {
	n := max - min + 1
	i := self.rng.Intn(n)
	return min + i
}

func (self *DunGen) isRectClear(room DRect) bool {
	for y := room.y; y <= room.bottom(); y++ {
		for x := room.x; x <= room.right(); x++ {
			if self.getCell(x, y) != TILE_UNUSED {
				return false
			}
		}
	}
	return true
}

func (self *DunGen) firstRoom() DRect {
	var rWidth = self.getRand(minRoomDim, maxRoomWidth)
	var rHeight = self.getRand(minRoomDim, maxRoomHeight)
	var xoff = self.rng.Intn(10) - 5
	var yoff = self.rng.Intn(5) - 2
	var x = self.xsize/2 - rWidth/2 + xoff
	var y = self.ysize/2 - rHeight/2 + yoff

	return DRect{
		x: x,
		y: y,
		w: rWidth,
		h: rHeight,
	}
}

func (self *DunGen) setWall(coord LCoord, direction int) {
	if coord.inShyBounds() && self.getCell(coord.x, coord.y) == TILE_UNUSED {
		for dir := 0; dir < 4; dir++ {
			if dir == direction {
				(self.walls[dir])[coord] = true
			} else {
				delete((self.walls[dir]), coord)
			}
		}
	}
}

func (self *DunGen) clearWalls(x int, y int) {
	for dir := 0; dir < 4; dir++ {
		delete(self.walls[dir], LCoord{x, y})
	}
}

func (self *DunGen) setRect(rect DRect) bool {
	if !self.isRectClear(rect) {
		return false
	}
	for y := rect.y; y < rect.bottom(); y++ {
		for x := rect.x; x < rect.right(); x++ {
			self.setCell(x, y, TILE_FLOOR)
		}
	}
	if rect.y > 1 {
		for x := rect.x; x < rect.right(); x++ {
			self.setWall(LCoord{x, rect.y - 1}, DIR_NORTH)
		}
	}
	if rect.bottom() < (self.ysize - 2) {
		for x := rect.x; x < rect.right(); x++ {
			self.setWall(LCoord{x, rect.bottom()}, DIR_SOUTH)
		}
	}
	if rect.x > 1 {
		for y := rect.y; y < rect.bottom(); y++ {
			self.setWall(LCoord{rect.x - 1, y}, DIR_WEST)
		}
	}
	if rect.right() < (self.xsize - 2) {
		for y := rect.y; y < rect.bottom(); y++ {
			self.setWall(LCoord{rect.right(), y}, DIR_EAST)
		}
	}
	self.objects = self.objects + 1
	return true
}

func (self *DunGen) setRoom(room DRect) bool {
	if !self.setRect(room) {
		return false
	}
	self.rooms[self.numRooms] = room
	self.numRooms += 1
	return true
}

func (self *DunGen) randomWall(dir int) (LCoord, bool) {
	wallMap := self.walls[dir]
	if len(wallMap) == 0 {
		return LCoord{}, false
	}
	n := self.rng.Intn(len(wallMap))
	i := 0
	var result LCoord
	for k := range wallMap {
		if i == n {
			result = k
			break
		}
		i++
	}
	return result, true
}

func (self *DunGen) pickStart() (LCoord, int) {
	var dir int
	var ok bool = false
	var coord LCoord
	for !ok {
		dir = self.rng.Intn(4)
		coord, ok = self.randomWall(dir)
	}
	return coord, dir
}

func (self *DunGen) newShyRect(x int, y int, w int, h int) DRect {
	x1, y1, w1, h1 := x, y, w, h
	if x == 0 {
		x1 = 1
		w1 = w - 1
	}
	if y == 0 {
		y1 = 1
		h1 = h - 1
	}
	if x1+w1 > self.xsize-2 {
		if self.xsize-2-x1 > 0 {
			w1 = self.xsize - 2 - x1
		}
	}
	if y1+h1 > self.ysize-2 {
		if self.ysize-2-y1 > 0 {
			h1 = self.ysize - 2 - y1
		}
	}
	return DRect{x1, y1, w1, h1}
}

func (self *DunGen) tryCorridor(coord LCoord, dir int) bool {
	corrLen := self.getRand(minCorridorDim, maxLengthCorridor)
	var rect DRect
	switch dir {
	case DIR_NORTH:
		rect = self.newShyRect(coord.x, coord.y-corrLen, 1, corrLen)
	case DIR_SOUTH:
		rect = self.newShyRect(coord.x, coord.y, 1, corrLen)
	case DIR_EAST:
		rect = self.newShyRect(coord.x, coord.y, corrLen, 1)
	case DIR_WEST:
		rect = self.newShyRect(coord.x-corrLen, coord.y, corrLen, 1)
	}
	result := self.setRect(rect)
	if result {
		self.setCell(coord.x, coord.y, TILE_FLOOR)
		switch dir {
		case DIR_NORTH, DIR_SOUTH:
			{
				self.clearWalls(coord.x+1, coord.y)
				self.clearWalls(coord.x-1, coord.y)
			}
		case DIR_WEST, DIR_EAST:
			{
				self.clearWalls(coord.x, coord.y+1)
				self.clearWalls(coord.x, coord.y-1)
			}
		}
	}
	return result
}

func (self *DunGen) tryRoom(coord LCoord, dir int) bool {
	h := self.getRand(minRoomDim, maxRoomHeight)
	w := self.getRand(minRoomDim, maxRoomWidth)

	var rect DRect
	switch dir {
	case DIR_NORTH:
		xoff := self.rng.Intn(w)
		rect = self.newShyRect(coord.x-xoff, coord.y-h, w, h)
	case DIR_SOUTH:
		xoff := self.rng.Intn(w)
		rect = self.newShyRect(coord.x-xoff, coord.y+1, w, h)
	case DIR_EAST:
		yoff := self.rng.Intn(h)
		rect = self.newShyRect(coord.x+1, coord.y-yoff, w, h)
	case DIR_WEST:
		yoff := self.rng.Intn(h)
		rect = self.newShyRect(coord.x-w, coord.y-yoff, w, h)
	}
	fmt.Println("rect:", rect, " dir: ", dir)
	result := self.setRect(rect)
	if result {
		self.setCell(coord.x, coord.y, TILE_FLOOR)
		switch dir {
		case DIR_NORTH, DIR_SOUTH:
			{
				self.clearWalls(coord.x+1, coord.y)
				self.clearWalls(coord.x-1, coord.y)
			}
		case DIR_WEST, DIR_EAST:
			{
				self.clearWalls(coord.x, coord.y+1)
				self.clearWalls(coord.x, coord.y-1)
			}
		}
	}
	return result
}

//func (self *DunGen) createDungeon(inx int, iny int, inobj int) bool {
func (self *DunGen) createDungeon() bool {
	self.setRoom(self.firstRoom())
	println(self.debugPrint())
	coord, dir := self.pickStart()
	self.tryCorridor(coord, dir)
	println(self.debugPrint())
	for self.objects < self.targetObj {
		coord, dir = self.pickStart()
		var added bool
		if self.rng.Intn(100) <= self.chanceRoom {
			added = self.tryRoom(coord, dir)
		} else {
			added = self.tryCorridor(coord, dir)
		}
		if added {
			println(self.debugPrint())
		}
	}
	return true
}
