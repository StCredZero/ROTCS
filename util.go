package main

import "math"

func abs(n int64) int64 {
	if n < 0 {
		return 0 - n
	} else {
		return n
	}
}

func manhattanDist(loc1 Coord, loc2 Coord) float64 {
	return float64(abs(loc1.x-loc2.x)) + float64(abs(loc1.y-loc2.y))
}

func distance(loc1 Coord, loc2 Coord) float64 {
	dx, dy := float64(abs(loc1.x-loc2.x)), float64(abs(loc1.y-loc2.y))
	return math.Sqrt(dx*dx + dy*dy)
}
