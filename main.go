package main

import (
	"flag"
	"fmt"
	//"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go/build"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"text/template"
	"time"
)

func IdGenerator(lastId uint32) chan (uint32) {
	next := make(chan uint32)
	id := lastId + 1
	go func() {
		for {
			next <- id
			id++
		}
	}()
	return next
}

type CstServer struct {
	// Registered connections.
	connections map[*connection]uint32

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	entityIdGen chan uint32

	world SubGrid
}

func updateLoc(move rune, loc Coord) Coord {
	if move == 'n' {
		return Coord{loc.x, loc.y - 1}
	} else if move == 's' {
		return Coord{loc.x, loc.y + 1}
	} else if move == 'w' {
		return Coord{loc.x - 1, loc.y}
	} else if move == 'e' {
		return Coord{loc.x + 1, loc.y}
	}
	return loc
}

func updateFn(grid GridKeeper, ntt Entity) {

	select {
	case moves := <-ntt.Connection.moveQueue:
		ntt.Moves = moves
	default:
	}

	var move rune = '0'
	for _, v := range ntt.Moves {
		move = v
		println(v)
		ntt.Moves = ntt.Moves[1:]
		break
	}

	newLoc := updateLoc(move, ntt.Location)
	fmt.Println(newLoc)
	grid.MoveEntity(ntt, newLoc)

	ntt.Connection.send <- []byte(grid.DisplayString())
}

func (srv *CstServer) Update(now time.Time) {
	srv.world.UpdateEntities(updateFn, srv)
}

func (srv *CstServer) run() {
	timer := time.Tick(125 * time.Millisecond)
	for {
		runtime.Gosched()
		select {
		case c := <-srv.register:
			println("starting register")
			var newId = <-srv.entityIdGen
			srv.connections[c] = newId

			newEntity := Entity{
				Id:         newId,
				Moves:      "",
				Connection: c,
			}
			newEntity, _ = srv.world.NewEntity(newEntity)
			fmt.Println("Initialized entity: ", newEntity)
		case c := <-srv.unregister:
			delete(srv.connections, c)
			println("closing-final")
			close(c.send)
		case now := <-timer:
			srv.Update(now)
		}
	}
}

func (srv *CstServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	c := &connection{
		send:      make(chan []byte, 256),
		moveQueue: make(chan string, 10),
		ws:        ws,
	}
	srv.register <- c
	defer func() { srv.unregister <- c }()
	go c.writer()
	c.reader(srv)
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	moveQueue chan string

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader(srv *CstServer) {
	for {
		_, message, err := c.ws.ReadMessage()
		//n := bytes.Index(message, []byte{0})
		s := string(message[:])
		print("got: ")
		println(s)
		if err != nil {
			break
		}
		c.moveQueue <- s
	}
	println("closing reader")
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	println("closing writer")
	c.ws.Close()
}

func defaultAssetPath() string {
	p, err := build.Default.Import("github.com/StCredZero/casterly", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

func homeHandler(c http.ResponseWriter, req *http.Request, homeTempl *template.Template) {
	homeTempl.Execute(c, req.Host)
}

func main() {

	flag.Parse()

	var srv = CstServer{
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]uint32),
	}

	srv.entityIdGen = IdGenerator(0)
	srv.world = SubGrid{
		GridCoord: Coord{0, 0},
		Grid:      make(map[Coord]uint32),
		Entities:  make(map[uint32]Entity),
	}
	go srv.run()

	var addr = flag.String("addr", ":8080", "http service address")
	var assets = flag.String("assets", defaultAssetPath(), "path to assets")
	var homeTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "index.html")))

	println(assets)
	println(homeTempl)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		srv.wsHandler(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(w, r, homeTempl)
	})

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
