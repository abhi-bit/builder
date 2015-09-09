package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var osBuildMapping = make(map[string]string)
var buildId int = 20
var mutex = &sync.Mutex{}
var ongoingJobs = make(chan int, 1)

func init() {
	osBuildMapping["centos6"] = "2222"
	osBuildMapping["centos7"] = "2223"
	osBuildMapping["ubuntu12"] = "2224"
	osBuildMapping["ubuntu14"] = "2225"
	osBuildMapping["debian7"] = "2226"
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to builder\n")
}

func getOSList(w http.ResponseWriter, r *http.Request) {
	OSList := make([]string, 0, len(osBuildMapping))
	for OS := range osBuildMapping {
		OSList = append(OSList, OS)
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := json.NewEncoder(w).Encode(OSList); err != nil {
		panic(err)
	}
}

func createBuild(w http.ResponseWriter, r *http.Request) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	if len(ongoingJobs) > 0 {
		fmt.Fprintf(w, "Build id: %d already running, please wait for 15 mins and retry\n", buildId)
		return
	} else {
		mutex.Lock()
		buildId = buildId + 1
		mutex.Unlock()
	}

	log.Println("Build id: ", buildId, r.URL.Path[1:])
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ongoingJobs <- 1
	vars := mux.Vars(r)
	os := vars["OS"]
	xmlFile := "toy/" + vars["xmlFile"]
	done := make(chan bool, 1)

	now := time.Now()

	cmd := exec.Command("./createBuild.sh", osBuildMapping[os],
		os, xmlFile, strconv.Itoa(buildId))

	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	err := cmd.Start()
	printError(err)

	fmt.Fprintf(w, "Started build for OS: %v xmlFile: %v, buildId: %d\n", os, xmlFile, buildId)

	go func() {
		for {
			select {
			case <-done:
				fmt.Fprintf(w, "Done\n")
				fmt.Fprintf(w, "Time elapsed: %s\n", time.Since(now))
				flusher.Flush()
				close(done)
				return
			default:
				fmt.Fprintf(w, ".")
				flusher.Flush()
			}
			time.Sleep(time.Second * 1)
		}
	}()
	cmd.Wait()
	done <- true
	<-ongoingJobs

	// Sleep to allow dump of timing stats
	time.Sleep(2 * time.Second)

	ext := ""
	if os == "centos6" || os == "centos7" {
		ext = "rpm"
	} else {
		ext = "deb"
	}

	// S3 download links:
	cbServer := `http://customers.couchbase.com.s3.amazonaws.com/couchbase/couchbase-server-1.7~toy-10` + strconv.Itoa(buildId) + `.0.0.0.x86_64.` + ext
	cbDebugServer := `http://customers.couchbase.com.s3.amazonaws.com/couchbase/couchbase-server-debug-1.7~toy-10` + strconv.Itoa(buildId) + `.0.0.0.x86_64.` + ext
	fmt.Fprintf(w, "S3 download links:")
	fmt.Fprintf(w, "%s\n", cbServer)
	fmt.Fprintf(w, "%s\n", cbDebugServer)
}

func main() {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", handler)
	router.HandleFunc("/OS", getOSList)
	router.HandleFunc("/build/{OS}/toy/{xmlFile}", createBuild)
	fmt.Println("Starting web service on 8080")
	http.ListenAndServe(":8080", router)
}

func printError(err error) {
	if err != nil {
		log.Println("Error received:", err)
	}
}
