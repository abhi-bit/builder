package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
)

var osBuildMapping = make(map[string]string)

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

	log.Println(r.URL.Path[1:])

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	vars := mux.Vars(r)
	os := vars["OS"]
	xmlFile := "toy/" + vars["xmlFile"]
	done := make(chan bool, 1)

	now := time.Now()

	cmd := exec.Command("./createBuild.sh", osBuildMapping[os], os, xmlFile)

	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	err := cmd.Start()
	printError(err)

	fmt.Fprintf(w, "Started build for OS: %v xmlFile: %v\n", os, xmlFile)

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

	// Sleep to allow dump timing stats
	time.Sleep(2 * time.Second)
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
