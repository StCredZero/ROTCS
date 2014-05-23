package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

var DungeonEntropy = DunGenEntropy([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 55, 13, 14, 15, 16})
var DungeonProto = DunGen{
	xsize:      subgrid_width,
	ysize:      subgrid_height,
	targetObj:  20,
	chanceRoom: 50,
}

var (
	TRACE   *log.Logger
	PROF    *log.Logger
	INFO    *log.Logger
	WARNING *log.Logger
	ERROR   *log.Logger

	traceFlag *bool
	profFlag  *bool
	infoFlag  *bool
	warnFlag  *bool
	errFlag   *bool
)

func LogTrace(args ...interface{}) {
	if *traceFlag {
		TRACE.Println(args...)
	}
}
func LogProfile(args ...interface{}) {
	if *profFlag {
		PROF.Println(args...)
	}
}
func LogInfo(args ...interface{}) {
	if *infoFlag {
		INFO.Println(args...)
	}
}
func LogWarn(args ...interface{}) {
	if *warnFlag {
		WARNING.Println(args...)
	}
}
func LogError(args ...interface{}) {
	if *errFlag {
		ERROR.Println(args...)
	}
}

func initLogging(traceHandle, profHandle, infoHandle, warningHandle, errorHandle io.Writer) {
	TRACE = log.New(traceHandle, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)
	PROF = log.New(profHandle, "PROFILE: ", log.Ldate|log.Ltime|log.Lshortfile)
	INFO = log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WARNING = log.New(warningHandle, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
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

	port := flag.String("port", ":8080", "http service port")
	assets := flag.String("assets", ".", "path to assets")
	htmlPath := filepath.Join(*assets, "static")

	traceFlag = flag.Bool("trace", false, "log trace messages")
	profFlag = flag.Bool("prof", true, "log profile messages")
	infoFlag = flag.Bool("info", true, "log info messages")
	warnFlag = flag.Bool("warn", true, "log warnings")
	errFlag = flag.Bool("error", true, "log errors")

	dev := flag.Bool("dev", false, "develop - run without TLS")

	flag.Parse()

	logPath := filepath.Join(*assets, "log")
	profPath := filepath.Join(*assets, "prof")

	logfile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	var writer, profWriter io.Writer
	writer = io.MultiWriter(logfile, os.Stdout)

	profile, err := os.OpenFile(profPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open profile log:", err)
	}
	if *profFlag {
		profWriter = profile
	} else {
		profWriter = ioutil.Discard
	}

	initLogging(
		logWriter(*traceFlag, writer),
		profWriter,
		logWriter(*infoFlag, writer),
		logWriter(*warnFlag, writer),
		logWriter(*errFlag, writer))

	log.SetOutput(writer)

	// Instantiate Server and start runLoop
	var srv = NewCstServer()
	go srv.runLoop()

	LogInfo("Port:", *port)
	LogInfo("Asset Path:", *assets)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		srv.wsHandler(w, r)
	})

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(htmlPath))))

	if *dev {
		if err := http.ListenAndServe(*port, nil); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	} else {
		if err := http.ListenAndServeTLS(*port, "etc/cert/certificate", "etc/cert/server.key", nil); err != nil {
			log.Fatal("ListenAndServeTLS:", err)
		}
	}
}
