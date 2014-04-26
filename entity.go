package main

type Entity struct {
	Id         uint32
	Location   Coord
	Connection *connection
	//MoveQueue [1024]chan rune
}

func (self *Entity) DisplayString() string {
	return self.Location.IndexString() + `:"@"`
}
