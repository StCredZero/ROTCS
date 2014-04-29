package main

import (
	"flag"
	"go/build"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

var debugFlag = true

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
	entropy := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	dgproto := DunGen{
		xsize:      subgrid_width,
		ysize:      subgrid_height,
		targetObj:  20,
		chanceRoom: 50,
	}
	dgcache := NewDunGenCache(1024, DunGenEntropy(entropy), dgproto)
	d1 := dgcache.DungeonAt(GridCoord{0, 0})
	println(d1.debugPrint())
	d2 := dgcache.DungeonAt(GridCoord{0, 1})
	println(d2.debugPrint())

	flag.Parse()

	var srv = CstServer{
		register:    make(chan *connection, 1000),
		unregister:  make(chan *connection, 1000),
		connections: make(map[*connection]EntityId),
		entityIdGen: EntityIdGenerator(0),
	}

	srv.world = SubGrid{
		GridCoord:   GridCoord{0, 0},
		Grid:        make(map[Coord]EntityId),
		Entities:    make(map[EntityId]Entity),
		ParentQueue: make(chan EntityId, (subgrid_width * subgrid_height)),
	}

	go srv.runLoop()

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
