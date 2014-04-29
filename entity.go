package main

type EntityId uint32

type Entity struct {
	Id         EntityId
	Location   Coord
	Moves      string
	Connection *connection
}

func (self *Entity) DisplayString() string {
	return self.Location.IndexString() + `:{"symbol":"@"}`
}

func EntityIdGenerator(lastId EntityId) chan (EntityId) {
	next := make(chan EntityId)
	id := lastId + 1
	go func() {
		for {
			next <- id
			id++
		}
	}()
	return next
}
