// Prg1: NPRG030
// ADS2: NTIN061
// Haskell: NAIL097
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/iamwave/samorozvrh/backend/cache"
	"github.com/iamwave/samorozvrh/backend/sisparse"
)

const FRONTEND_DIR = "frontend/dist"
const TIME_PER_QUERY = 3000

var rootDir string

func sisQueryHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Path[len("/sisquery/"):]
	query = strings.Trim(query, " ")

	if len(query) > 32 {
		fmt.Fprintf(w, `{"error":"The query is too long"}`)
		return
	}

	re := regexp.MustCompile(`\w*$`)

	if !re.MatchString(query) {
		fmt.Fprintf(w, `{"error":"Query must contain only alphanumeric characters"}`)
		return
	}
	var res string
	var err error

	query = strings.ToUpper(query)
	log.Printf("Sisquery (from %s): %s", r.RemoteAddr, ellipsis(query, 10))
	cacheName := []string{"courses", query}

	if cache.Has(cacheName) {
		log.Println("  (using cache)")
		res, err = cache.Get(cacheName)
	} else {
		log.Println("  (querying)")
		var events [][]sisparse.Event
		events, err = sisparse.GetCourseEvents(query)
		if err == nil {
			var s []byte
			s, err = json.Marshal(events)
			if err == nil {
				res = fmt.Sprintf(`{"data":%s}`, string(s))
				err = cache.Set(cacheName, res)
			}
		}
	}

	if err != nil {
		log.Printf("Sisquery error: %s", err)
		fmt.Fprintf(w, `{"error":"%s"}`, err)
	} else {
		log.Printf(`Sisquery answer: %s`, ellipsis(res, 30))
		fmt.Fprint(w, res)
	}
}

func solverQueryHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		log.Printf("Solverquery error: %s", err)
		fmt.Fprintf(w, `{"error":"%s"}`, err)
		return
	}
	if len(body) == 0 {
		log.Printf("Solverquery error: %s", err)
		fmt.Fprint(w, `{"error":"Request body must be non-empty"}`)
		return
	}

	log.Printf("Solverquery (from %s): %s\n", r.RemoteAddr, ellipsis(string(body), 30))
	res, err := Solve(body, TIME_PER_QUERY)
	if err != nil {
		log.Printf("Solverquery error: %s", err)
		fmt.Fprintf(w, `{"error":"%s"}`, err)
	} else {
		log.Printf("Solverquery answer: %s", ellipsis(string(res), 30))
		fmt.Fprint(w, string(res))
	}
}

func main() {
	rdir := flag.String("rootdir", ".", "path to Samorozvrh root directory")
	port := flag.Int("port", 8080, "port on which to start the server")
	flag.Parse()

	rootDir = *rdir
	cache.SetRootDir(rootDir)

	http.HandleFunc("/sisquery/", sisQueryHandler)
	http.HandleFunc("/solverquery/", solverQueryHandler)

	fs := http.FileServer(http.Dir(path.Join(rootDir, FRONTEND_DIR)))
	http.Handle("/", fs)

	logFilePath := "log.txt"
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		log.Fatalf("Could not open log file: %s\n", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	log.Printf("Listening on: %d", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

func ellipsis(s string, n int) string {
	if len(s) < n {
		return s
	} else {
		return s[:n] + "..."
	}
}
