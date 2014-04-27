package main

import "math/rand"

const TILE_UNUSED = 0
const TILE_WALL = 1
const TILE_FLOOR = 2
const maxRoomLength = 28
const maxRoomHeight = 14
const maxLengthCorridor = 16

type DunGen struct {
	xsize   int
	ysize   int
	objects int

	chanceRoom     int
	chanceCorridor int

	dungeon_map [subgrid_width * subgrid_height]int

	randomizer *rand.Rand
}

func (self *DunGen) initialize() *DunGen {
	self.xsize = subgrid_width
	self.ysize = subgrid_height
	self.objects = 30
	self.chanceRoom = 75
	self.chanceCorridor = 25
	//self.dungeon_map = make([]int, subgrid_width*subgrid_height)
	return self
}

func (self *DunGen) setCell(x int, y int, celltype int) {
	if x >= 0 && x < self.xsize && y >= 0 && y < self.ysize {
		self.dungeon_map[x+self.xsize*y] = celltype
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
	return (self.getCell(x, y) == TILE_FLOOR)
}

func (self *DunGen) getRand(min int, max int) int {
	n := max - min + 1
	i := self.randomizer.Intn(n)
	if i < 0 {
		i = -1
	}
	return min + i
}

func (self *DunGen) makeCorridor(x int, y int, length int, direction int) bool {
	c_len := self.getRand(2, length)
	var dir int = 0

	if direction > 0 && direction < 4 {
		dir = direction
	}

	var xtemp int = 0
	var ytemp int = 0

	switch dir {
	case 0: //north
		{
			//check if there's enough space for the corridor
			//start with checking it's not out of the boundaries
			if x < 0 || x > self.xsize {
				return false
			} else {
				xtemp = x
			}

			//same thing here, to make sure it's not out of the boundaries
			for ytemp = y; ytemp > (y - c_len); ytemp-- {
				if ytemp < 0 || ytemp > self.ysize {
					return false
				}
				if self.getCell(xtemp, ytemp) != TILE_UNUSED {
					return false
				}
			}

			//if we're still here, let's start building
			for ytemp = y; ytemp > (y - c_len); ytemp-- {
				self.setCell(xtemp, ytemp, TILE_FLOOR)
			}
		}
	case 1: //east
		{
			if y < 0 || y > self.ysize {
				return false
			} else {
				ytemp = y
			}

			for xtemp = x; xtemp < (x + c_len); xtemp++ {
				if xtemp < 0 || xtemp > self.xsize {
					return false
				}
				if self.getCell(xtemp, ytemp) != TILE_UNUSED {
					return false
				}
			}

			for xtemp = x; xtemp < (x + c_len); xtemp++ {
				self.setCell(xtemp, ytemp, TILE_FLOOR)
			}
		}
	case 2: //south
		{
			if x < 0 || x > self.xsize {
				return false
			} else {
				xtemp = x
			}
			for ytemp = y; ytemp < (y + c_len); ytemp++ {
				if ytemp < 0 || ytemp > self.ysize {
					return false
				}
				if self.getCell(xtemp, ytemp) != TILE_UNUSED {
					return false
				}
			}

			for ytemp = y; ytemp < (y + c_len); ytemp++ {
				self.setCell(xtemp, ytemp, TILE_FLOOR)
			}
		}
	case 3: //west
		{
			if ytemp < 0 || ytemp > self.ysize {
				return false
			} else {
				ytemp = y
			}

			for xtemp = x; xtemp > (x - c_len); xtemp-- {
				if xtemp < 0 || xtemp > self.xsize {
					return false
				}
				if self.getCell(xtemp, ytemp) != TILE_UNUSED {
					return false
				}
			}

			for xtemp = x; xtemp > (x - c_len); xtemp-- {
				self.setCell(xtemp, ytemp, TILE_FLOOR)
			}
		}
	}
	return true
}

func (self *DunGen) makeRoom(x int, y int, xlength int, ylength int, direction int) bool {
	xlen := self.getRand(4, xlength)
	ylen := self.getRand(4, ylength)

	var dir int = 0

	if direction > 0 && direction < 4 {
		dir = direction
	}

	switch dir {
	case 0: //north
		{
			//Check if there's enough space left for it
			for ytemp := y; ytemp > (y - ylen); ytemp-- {
				if ytemp < 0 || ytemp >= self.ysize {
					return false
				}
				for xtemp := (x - xlen/2); xtemp < (x + (xlen+1)/2); xtemp++ {
					if xtemp < 0 || xtemp >= self.xsize {
						return false
					}
					if self.getCell(xtemp, ytemp) != TILE_UNUSED {
						return false //no space left...
					}
				}
				for ytemp := y; ytemp > (y - ylen); ytemp-- {
					for xtemp := (x - xlen/2); xtemp < (x + (xlen+1)/2); xtemp++ {
						//start with the walls
						if xtemp == (x - xlen/2) {
							self.setCell(xtemp, ytemp, TILE_WALL)
						} else if xtemp == (x + (xlen-1)/2) {
							self.setCell(xtemp, ytemp, TILE_WALL)
						} else if ytemp == y {
							self.setCell(xtemp, ytemp, TILE_WALL)
						} else if ytemp == (y - ylen + 1) {
							self.setCell(xtemp, ytemp, TILE_WALL)
							//and then fill with the TILE_FLOOR
						} else {
							self.setCell(xtemp, ytemp, TILE_FLOOR)
						}
					}
				}
			}
		}
	case 1: //east
		{
			for ytemp := (y - ylen/2); ytemp < (y + (ylen+1)/2); ytemp++ {
				if ytemp < 0 || ytemp >= self.ysize {
					return false
				}
				for xtemp := x; xtemp < (x + xlen); xtemp++ {
					if xtemp < 0 || xtemp >= self.xsize {
						return false
					}
					if self.getCell(xtemp, ytemp) != TILE_UNUSED {
						return false
					}
				}
			}
			for ytemp := (y - ylen/2); ytemp < (y + (ylen+1)/2); ytemp++ {
				for xtemp := x; xtemp < (x + xlen); xtemp++ {
					if xtemp == x {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if xtemp == (x + xlen - 1) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == (y - ylen/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == (y + (ylen-1)/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else {
						self.setCell(xtemp, ytemp, TILE_FLOOR)
					}
				}
			}
		}
	case 2: //south
		{
			for ytemp := y; ytemp < (y + ylen); ytemp++ {
				if ytemp < 0 || ytemp >= self.ysize {
					return false
				}
				for xtemp := (x - xlen/2); xtemp < (x + (xlen+1)/2); xtemp++ {
					if xtemp < 0 || xtemp >= self.xsize {
						return false
					}
					if self.getCell(xtemp, ytemp) != TILE_UNUSED {
						return false
					}
				}
			}
			for ytemp := y; ytemp < (y + ylen); ytemp++ {
				for xtemp := (x - xlen/2); xtemp < (x + (xlen+1)/2); xtemp++ {
					if xtemp == (x - xlen/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if xtemp == (x + (xlen-1)/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == y {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == (y + ylen - 1) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else {
						self.setCell(xtemp, ytemp, TILE_FLOOR)
					}
				}
			}
		}
	case 3: //west
		{
			for ytemp := (y - ylen/2); ytemp < (y + (ylen+1)/2); ytemp++ {
				if ytemp < 0 || ytemp >= self.ysize {
					return false
				}
				for xtemp := x; xtemp >= (x - xlen); xtemp-- {
					if xtemp < 0 || xtemp > self.xsize {
						return false
					}
					if self.getCell(xtemp, ytemp) != TILE_UNUSED {
						return false
					}
				}
			}
			for ytemp := (y - ylen/2); ytemp < (y + (ylen+1)/2); ytemp++ {
				for xtemp := x; xtemp > (x - xlen); xtemp-- {
					if xtemp == x {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if xtemp == (x - xlen + 1) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == (y - ylen/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else if ytemp == (y + (ylen-1)/2) {
						self.setCell(xtemp, ytemp, TILE_WALL)
					} else {
						self.setCell(xtemp, ytemp, TILE_FLOOR)
					}
				}
			}
		}

	} //switch
	return true
}

//func (self *DunGen) createDungeon(inx int, iny int, inobj int) bool {
func (self *DunGen) createDungeon() bool {
	//start with making the "standard stuff" on the map
	for y := 0; y < self.ysize; y++ {
		for x := 0; x < self.xsize; x++ {
			self.setCell(x, y, TILE_UNUSED)
		}
	}
	//start with making a room in the middle, which we can start building upon
	self.makeRoom(self.xsize/2, self.ysize/2, maxRoomLength, maxRoomHeight, self.getRand(0, 3))

	var currentFeatures int = 1
	for countingTries := 0; countingTries < 1000; countingTries++ {
		//check if we've reached our quota
		if currentFeatures == self.objects {
			break
		}
		//start with a random wall
		var newx int = 0
		var xmod int = 0
		var newy int = 0
		var ymod int = 0
		var validTile int = -1
		//1000 chances to find a suitable object (room or corridor)..
		for testing := 0; testing < 1000; testing++ {
			newx = self.getRand(1, self.xsize-2)
			newy = self.getRand(1, self.ysize-2)
			validTile = -1
			if self.getCell(newx, newy) == TILE_WALL || self.getCell(newx, newy) == TILE_FLOOR {
				//check if we can reach the place
				if self.getCell(newx, newy+1) == TILE_FLOOR {
					validTile = 0
					xmod = 0
					ymod = -1
				} else if self.getCell(newx-1, newy) == TILE_FLOOR {
					validTile = 1
					xmod = +1
					ymod = 0
				} else if self.getCell(newx, newy-1) == TILE_FLOOR {
					validTile = 2
					xmod = 0
					ymod = +1
				} else if self.getCell(newx+1, newy) == TILE_FLOOR {
					validTile = 3
					xmod = -1
					ymod = 0
				}
				//if we can, jump out of the loop and continue with the rest
				if validTile > -1 {
					break
				}
			}
		}
		if validTile > -1 {
			//choose what to build now at our newly found place, and at what direction
			feature := self.getRand(0, 100)
			if feature <= self.chanceRoom { //a new room
				if self.makeRoom((newx + xmod), (newy + ymod), maxRoomLength, maxRoomHeight, validTile) {
					currentFeatures++ //add to our quota
					//then we mark the wall opening with a door
					self.setCell(newx, newy, TILE_FLOOR)
					//clean up infront of the door so we can reach it
					self.setCell((newx + xmod), (newy + ymod), TILE_FLOOR)
				}
			} else if feature >= self.chanceRoom { //new corridor
				if self.makeCorridor((newx + xmod), (newy + ymod), maxLengthCorridor, validTile) {
					//same thing here, add to the quota and a door
					currentFeatures++
					self.setCell(newx, newy, TILE_FLOOR)
				}
			}
		}
	}
	return true
}
