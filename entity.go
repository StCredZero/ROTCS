package main

type Entity struct {
	Id         uint32
	Location   Coord
	Moves      string
	Connection *connection
}

func (self *Entity) DisplayString() string {
	return self.Location.IndexString() + `:{"symbol":"@"}`
}
