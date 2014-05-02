package main

func abs(n int64) int64 {
	if n < 0 {
		return 0 - n
	} else {
		return n
	}
}

func manhattanDist(loc1 Coord, loc2 Coord) int64 {
	return abs(loc1.x-loc2.x) + abs(loc1.y-loc2.y)
}
