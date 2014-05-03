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

func expandGrids(grids *(map[GridCoord]bool)) *(map[GridCoord]bool) {
	newgr := make(map[GridCoord]bool)
	for gcoord, _ := range *grids {
		for _, gc := range gcoord.Expansion() {
			newgr[gc] = true
		}
	}
	return &newgr
}

func copyGrids(grids *(map[GridCoord]bool)) *(map[GridCoord]bool) {
	newgr := make(map[GridCoord]bool)
	for gc, _ := range *grids {
		newgr[gc] = true
	}
	return &newgr
}

func subtractGrids(gr1, gr2 *(map[GridCoord]bool)) {
	for gc, _ := range *gr2 {
		delete(*gr1, gc)
	}
}

func intersectGrids(gr1, gr2 *(map[GridCoord]bool)) {
	for gc, _ := range *gr1 {
		if _, present := (*gr2)[gc]; !present {
			delete(*gr1, gc)
		}
	}
}
