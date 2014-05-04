package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"sort"
)

const TileUnused = 0
const TileWall = 1
const TileFloor = 2
const TileUnpass = 3
const TileCorridor = 4
const TileDoor = 5

const minRoomDim = 2
const maxRoomWidth = 28
const maxRoomHeight = 14
const minCorridorDim = 2
const maxLengthCorridor = 16
const maxRooms = 100

const North = 0
const South = 1
const East = 2
const West = 3

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
	passaged   bool

	dungeon_map [subgrid_width * subgrid_height]int8

	walls [4](map[LCoord]bool)
	rooms []DRect

	passageSouth    LCoord
	passageEast     LCoord
	passageNorthEnd int
	passageWestEnd  int

	rng *rand.Rand
}

func NewDunGen(proto *DunGen) *DunGen {
	return &DunGen{
		xsize:      proto.xsize,
		ysize:      proto.ysize,
		targetObj:  proto.targetObj,
		chanceRoom: proto.chanceRoom,
	}
}

func (self *DunGen) setCell(x int, y int, value int8) {
	if x >= 0 && x < self.xsize && y >= 0 && y < self.ysize {
		self.dungeon_map[x+(self.xsize*y)] = value
		self.clearWalls(x, y)
	}
}

func (self *DunGen) getCell(x int, y int) int8 {
	if x >= 0 && x < self.xsize && y >= 0 && y < self.ysize {
		return self.dungeon_map[x+(self.xsize*y)]
	} else {
		return TileUnused
	}
}

func (self *DunGen) isWalkable(x int, y int) bool {
	return ((self.getCell(x, y) == TileFloor) ||
		(self.getCell(x, y) == TileCorridor) ||
		(self.getCell(x, y) == TileDoor))
}

func (self *DunGen) debugPrint() string {
	var buffer bytes.Buffer
	for y := 0; y < self.ysize; y++ {
		for x := 0; x < self.xsize; x++ {
			if self.passageSouth.x == x && self.passageSouth.y == y {
				buffer.WriteString("V")
			} else if self.passageEast.x == x && self.passageEast.y == y {
				buffer.WriteString("}")
			} else if (self.walls[North])[LCoord{x, y}] {
				buffer.WriteString("N")
			} else if (self.walls[South])[LCoord{x, y}] {
				buffer.WriteString("S")
			} else if (self.walls[West])[LCoord{x, y}] {
				buffer.WriteString("W")
			} else if (self.walls[East])[LCoord{x, y}] {
				buffer.WriteString("E")
			} else if self.isWalkable(x, y) {
				buffer.WriteString(".")
			} else {
				buffer.WriteString("0")
			}
		}
		buffer.WriteString("\n")
	}
	if debugFlag {
		for i := 0; i < self.numRooms; i++ {
			fmt.Print(self.rooms[i])
			println()
		}
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
			if self.getCell(x, y) != TileUnused {
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
	if coord.inBounds() && self.getCell(coord.x, coord.y) == TileUnused {
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
			self.setCell(x, y, TileFloor)
		}
	}
	if rect.y > 1 {
		for x := rect.x; x < rect.right(); x++ {
			self.setWall(LCoord{x, rect.y - 1}, North)
		}
	}
	if rect.bottom() < (self.ysize - 2) {
		for x := rect.x; x < rect.right(); x++ {
			self.setWall(LCoord{x, rect.bottom()}, South)
		}
	}
	if rect.x > 1 {
		for y := rect.y; y < rect.bottom(); y++ {
			self.setWall(LCoord{rect.x - 1, y}, West)
		}
	}
	if rect.right() < (self.xsize - 2) {
		for y := rect.y; y < rect.bottom(); y++ {
			self.setWall(LCoord{rect.right(), y}, East)
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
	self.numRooms = self.numRooms + 1
	return true
}

func (self *DunGen) randomWall(dir int) (LCoord, bool) {
	wallMap := self.walls[dir]
	if len(wallMap) == 0 {
		return LCoord{}, false
	}
	n := len(wallMap)
	keys := make(SortableLCoords, n, n)
	i := 0
	for k, _ := range wallMap {
		keys[i] = k
		i++
	}
	sort.Sort(keys)
	return keys[self.rng.Intn(n)], true
}

func (self *DunGen) pickStartDir(dir int) LCoord {
	var ok bool = false
	var coord LCoord
	for !ok {
		coord, ok = self.randomWall(dir)
	}
	return coord
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
	if x <= 0 {
		x1, w1 = 1, w-1
	}
	if y <= 0 {
		y1, h1 = 1, h-1
	}
	if x1+w1 > self.xsize-1 {
		if self.xsize-1-x1 > 0 {
			w1 = self.xsize - 1 - x1
		}
	}
	if y1+h1 > self.ysize-1 {
		if self.ysize-1-y1 > 0 {
			h1 = self.ysize - 1 - y1
		}
	}
	return DRect{x1, y1, w1, h1}
}

func (self *DunGen) tryCorridor(coord LCoord, dir int) bool {
	corrLen := self.getRand(minCorridorDim, maxLengthCorridor)
	var rect DRect
	switch dir {
	case North:
		rect = self.newShyRect(coord.x, coord.y-corrLen, 1, corrLen)
	case South:
		rect = self.newShyRect(coord.x, coord.y, 1, corrLen)
	case East:
		rect = self.newShyRect(coord.x, coord.y, corrLen, 1)
	case West:
		rect = self.newShyRect(coord.x-corrLen, coord.y, corrLen, 1)
	}
	result := self.setRect(rect)
	if result {
		self.setCell(coord.x, coord.y, TileFloor)
		switch dir {
		case North, South:
			self.clearWalls(coord.x+1, coord.y)
			self.clearWalls(coord.x-1, coord.y)
		case West, East:
			self.clearWalls(coord.x, coord.y+1)
			self.clearWalls(coord.x, coord.y-1)
		}
	}
	return result
}

func (self *DunGen) tryRoom(coord LCoord, dir int) bool {
	h := self.getRand(minRoomDim, maxRoomHeight)
	w := self.getRand(minRoomDim, maxRoomWidth)

	var rect DRect
	switch dir {
	case North:
		xoff := self.rng.Intn(w)
		rect = self.newShyRect(coord.x-xoff, coord.y-h, w, h)
	case South:
		xoff := self.rng.Intn(w)
		rect = self.newShyRect(coord.x-xoff, coord.y+1, w, h)
	case East:
		yoff := self.rng.Intn(h)
		rect = self.newShyRect(coord.x+1, coord.y-yoff, w, h)
	case West:
		yoff := self.rng.Intn(h)
		rect = self.newShyRect(coord.x-w, coord.y-yoff, w, h)
	}

	result := self.setRoom(rect)
	if result {
		self.setCell(coord.x, coord.y, TileFloor)
		switch dir {
		case North, South:
			self.clearWalls(coord.x+1, coord.y)
			self.clearWalls(coord.x-1, coord.y)
		case West, East:
			self.clearWalls(coord.x, coord.y+1)
			self.clearWalls(coord.x, coord.y-1)
		}
	}
	return result
}

type DunGenEntropy []byte

func (bytes DunGenEntropy) WriteTo(b *bytes.Buffer) {
	b.Write(bytes)
}

func (self *DunGen) createDungeon(gridCoord GridCoord, entropy DunGenEntropy) {

	var buffer bytes.Buffer
	gridCoord.WriteTo(&buffer)
	entropy.WriteTo(&buffer)
	h := sha1.New()
	bs := h.Sum(buffer.Bytes())
	var newSeed, i uint64
	for i = 0; i < 20; i++ {
		newSeed ^= uint64(bs[i]) << (i % 8)
	}
	self.rng = rand.New(rand.NewSource(int64(newSeed)))

	self.rooms = make([]DRect, self.targetObj, self.targetObj)
	for i := 0; i < 4; i++ {
		self.walls[i] = make(map[LCoord]bool)
	}

	self.setRoom(self.firstRoom())
	for self.objects < self.targetObj {
		coord, dir := self.pickStart()
		if self.rng.Intn(100) <= self.chanceRoom {
			self.tryRoom(coord, dir)
		} else {
			self.tryCorridor(coord, dir)
		}
	}

	self.passageSouth = self.pickStartDir(South)
	self.passageEast = self.pickStartDir(East)
	self.passageNorthEnd = self.getRand(1, self.ysize-2)
	self.passageWestEnd = self.getRand(1, self.xsize-2)

	newRooms := make([]DRect, self.numRooms)
	copy(newRooms, self.rooms[:self.numRooms])
	self.rooms = newRooms
	for i := 0; i < 4; i++ {
		self.walls[i] = nil
	}
	self.rng = nil
}

func (self *DunGen) makePassages(northDg *DunGen, westDg *DunGen) {
	if !self.passaged {
		nx := northDg.passageSouth.x
		for y := 0; y <= self.passageNorthEnd; y++ {
			self.setCell(nx, y, TileFloor)
		}
		for y1 := self.passageSouth.y; y1 < self.ysize; y1++ {
			self.setCell(self.passageSouth.x, y1, TileFloor)
		}
		wy := westDg.passageEast.y
		for x := 0; x <= self.passageWestEnd; x++ {
			self.setCell(x, wy, TileFloor)
		}
		for x1 := self.passageEast.x; x1 < self.xsize; x1++ {
			self.setCell(x1, self.passageEast.y, TileFloor)
		}
		self.passaged = true
	}
}

func (self *DunGen) TileAt(lcoord LCoord) int8 {
	return self.dungeon_map[lcoord.x+(lcoord.y*self.xsize)]
}
