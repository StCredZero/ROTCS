package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
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
	p, err := build.Default.Import("github.com/StCredZero/ROTCS", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

func homeHandler(c http.ResponseWriter, req *http.Request, homeTempl *template.Template) {
	homeTempl.Execute(c, req.Host)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//runtime.GOMAXPROCS(1)

	flag.Parse()

	// Instantiate Server and start runLoop
	var srv = NewCstServer()
	go srv.runLoop()

	var addr = flag.String("addr", ":8080", "http service address")
	var assets = flag.String("assets", defaultAssetPath(), "path to assets")
	var htmlPath = filepath.Join(*assets, "static")
	var homeTempl = template.Must(template.ParseFiles(filepath.Join(htmlPath, "index.html")))

	fmt.Println(*addr)
	fmt.Println(*assets)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		srv.wsHandler(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(w, r, homeTempl)
	})

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(htmlPath))))

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
