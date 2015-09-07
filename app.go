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

func osPortMapping() {
	osBuildMapping["centos6"] = "2222"
	osBuildMapping["centos7"] = "2223"
	osBuildMapping["ubuntu12"] = "2224"
	osBuildMapping["ubuntu14"] = "2225"
	osBuildMapping["debian7"] = "2226"
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to builder")
}

func getOSList(w http.ResponseWriter, r *http.Request) {
	OSList := make([]string, 0, len(osBuildMapping))
	for OS := range osBuildMapping {
		OSList = append(OSList, OS)
	}

	js, err := json.Marshal(OSList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func createBuild(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	os := vars["OS"]
	xmlFile := "toy/" + vars["xmlFile"]

	cmd := exec.Command("./createBuild.sh",
		osBuildMapping[os], os, xmlFile)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	err := cmd.Start()
	printError(err)

	ticker := time.NewTicker(time.Second)
	go func(ticker *time.Ticker) {
		now := time.Now()
		for _ = range ticker.C {
			printOutput(
				[]byte(fmt.Sprintf("%s", time.Since(now))),
			)
		}
	}(ticker)

	cmd.Wait()
	printOutput(
		[]byte(fmt.Sprintf("%s", cmdOutput.Bytes())),
	)

	out, err := exec.Command("./createBuild.sh",
		osBuildMapping[os], os, xmlFile).Output()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("output:", string(out))
	}
	log.Printf("Successfully created build for os: %s xml: %s", os, xmlFile)

	fmt.Fprintf(w, "Started build for OS: %v xmlFile: %v", os, xmlFile)
}

func main() {
	//Construct OS names to docker build mapping
	osPortMapping()

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", handler)
	router.HandleFunc("/OS", getOSList)
	router.HandleFunc("/build/{OS}/toy/{xmlFile}", createBuild)
	fmt.Println("Starting web service on 8080")
	http.ListenAndServe(":8080", router)
}

func printError(err error) {
	log.Println(err)
}

func printOutput(output []byte) {
	log.Println(string(output))
}
