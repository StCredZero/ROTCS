package main

import (
	"flag"
	//"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

var (
	TRACE   *log.Logger
	INFO    *log.Logger
	WARNING *log.Logger
	ERROR   *log.Logger
)

var DungeonEntropy = DunGenEntropy([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 55, 13, 14, 15, 16})
var DungeonProto = DunGen{
	xsize:      subgrid_width,
	ysize:      subgrid_height,
	targetObj:  20,
	chanceRoom: 50,
}

func initLogging(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	TRACE = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	INFO = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	WARNING = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	ERROR = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
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

func logWriter(shouldWrite bool, file io.Writer) io.Writer {
	if shouldWrite {
		return file
	}
	return ioutil.Discard
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	//runtime.GOMAXPROCS(1)

	flag.Parse()

	var addr = flag.String("addr", ":8080", "http service address")
	var assets = flag.String("assets", defaultAssetPath(), "path to assets")
	var htmlPath = filepath.Join(*assets, "static")
	var homeTempl = template.Must(template.ParseFiles(filepath.Join(htmlPath, "index.html")))

	logPath := filepath.Join(*assets, "log")
	trace := flag.Bool("trace", false, "log trace messages")
	info := flag.Bool("info", true, "log info messages")
	warn := flag.Bool("warn", true, "log warnings")
	errf := flag.Bool("error", true, "log errors")

	logfile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	multi := io.MultiWriter(logfile, os.Stderr)
	initLogging(
		logWriter(*trace, multi),
		logWriter(*info, multi),
		logWriter(*warn, multi),
		logWriter(*errf, multi))

	// Instantiate Server and start runLoop
	var srv = NewCstServer()
	go srv.runLoop()

	INFO.Println("Port:", *addr)
	INFO.Println("Asset Path:", *assets)

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
