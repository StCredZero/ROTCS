package main

import (
	"container/heap"
	//"fmt"
)

type NodeF struct {
	coord    Coord
	priority int64
	index    int
}

type NodeFPQ []*NodeF

func (self NodeFPQ) Len() int {
	return len(self)
}
func (self NodeFPQ) Less(i, j int) bool {
	return self[i].priority < self[j].priority
}
func (self NodeFPQ) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
	self[i].index = i
	self[j].index = j
}
func (self *NodeFPQ) Push(x interface{}) {
	n := len(*self)
	item := x.(*NodeF)
	item.index = n
	*self = append(*self, item)
}

func (self *NodeFPQ) Pop() interface{} {
	old := *self
	n := len(old)
	item := old[n-1]
	item.index = -1
	*self = old[0 : n-1]
	return item
}

func (pq *NodeFPQ) update(item *NodeF, coord Coord, priority int64) {
	heap.Remove(pq, item.index)
	item.coord = coord
	item.priority = priority
	heap.Push(pq, item)
}

func reconstructPath(cameFrom map[Coord]Coord, current Coord) []Coord {
	if next, present := cameFrom[current]; present {
		p := reconstructPath(cameFrom, next)
		return append(p, current)
	} else {
		return []Coord{}
	}
}

func lowestOpenFScore(openSet *(map[Coord]bool), fScore *NodeFPQ) (Coord, bool) {
	if len(*fScore) == 0 {
		return Coord{}, false
	}
	p1 := heap.Pop(fScore).(*NodeF)
	if (*openSet)[p1.coord] {
		return p1.coord, true
	} else {
		c, ok := lowestOpenFScore(openSet, fScore)
		heap.Push(fScore, p1)
		if ok {
			return c, true
		} else {
			return Coord{}, false
		}
	}
}

func setFScore(fScore *NodeFPQ, fIndex *(map[Coord]*NodeF), coord Coord, score int64) {
	fs, present := (*fIndex)[coord]
	if present {
		fScore.update(fs, coord, score)
	} else {
		fnew := &NodeF{coord, score, -1}
		(*fIndex)[coord] = fnew
		heap.Push(fScore, fnew)
	}
}

func astarSearch(
	heuristic func(Coord, Coord) int64,
	openAt func(Coord) bool,
	neighbors func(Coord) []Coord,
	start Coord,
	goal Coord,
	limit int) ([]Coord, bool) {

	closedSet := make(map[Coord]bool)
	openSet := make(map[Coord]bool)
	cameFrom := make(map[Coord]Coord)
	gScore := make(map[Coord]int64)
	fScore := make(NodeFPQ, 0, 1)
	fIndex := make(map[Coord]*NodeF)

	openSet[start] = true
	gScore[start] = 0
	setFScore(&fScore, &fIndex, start, heuristic(start, goal))

	for n := 0; len(openSet) > 0 && n < limit; n++ {
		// current := the node in openset having the lowest f_score[]
		current, ok := lowestOpenFScore(&openSet, &fScore)
		if !ok {
			return []Coord{}, false
		}
		if current == goal {
			return reconstructPath(cameFrom, goal), true
		}
		delete(openSet, current)
		closedSet[current] = true
		for _, neighbor := range neighbors(current) {
			if !openAt(neighbor) || closedSet[neighbor] {
				continue
			}
			tentativeg := gScore[current] + heuristic(current, neighbor)
			if (!openSet[neighbor]) || (tentativeg < gScore[neighbor]) {
				cameFrom[neighbor] = current
				gScore[neighbor] = tentativeg
				//fScore[neighbor] = gScore[neighbor] + heuristic(neighbor, goal)
				setFScore(&fScore, &fIndex, neighbor, tentativeg+heuristic(neighbor, goal))
				if !openSet[neighbor] {
					openSet[neighbor] = true
				}
			}
		}
	}
	return []Coord{}, false
}
