package main

import (
	"flag"
	"go/build"
	"log"
	"net/http"
	"path/filepath"
	//"runtime"
	"text/template"

	//"container/heap"
	//"fmt"
)

var debugFlag = false

var DungeonEntropy = DunGenEntropy([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 55, 13, 14, 15, 16})
var DungeonProto = DunGen{
	xsize:      subgrid_width,
	ysize:      subgrid_height,
	targetObj:  20,
	chanceRoom: 50,
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

/*var myarray = [][]int{
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 1, 0, 0, 0, 0, 1, 1, 1, 1, 0},
	{0, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0},
	{0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0},
	{0, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0},
	{0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0},
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

func testOpen(coord Coord) bool {
	if coord.x >= 0 && coord.y >= 0 && coord.x < 10 && coord.y < 10 {
		return myarray[coord.y][coord.x] != 0
	}
	return false
}*/

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())

	/*result, ok := astarSearch(manhattanDist, testOpen, neighbors4, Coord{1, 1}, Coord{1, 8}, 100)
	if ok {
		fmt.Println(result)
	} else {
		println("not found")
	}*/

	flag.Parse()

	// Instantiate Server and start runLoop
	var srv = NewCstServer()
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
