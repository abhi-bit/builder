package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var osBuildMapping = map[string]string{
	"centos6":  "2222",
	"centos7":  "2223",
	"ubuntu12": "2224",
	"ubuntu14": "2225",
	"debian7":  "2226",
}
var mutex = &sync.Mutex{}
var startTime = time.Now()
var wg = &sync.WaitGroup{}

var options struct {
	basedir string
	logfile string
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
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

func upTime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text;charset=UTF-8")
	fmt.Fprintf(w, "ticking for: %v\n", time.Since(startTime))
}

func getConfiguration(w http.ResponseWriter, r *http.Request) {
	data, err := marshalConfig()
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	fmt.Fprintf(w, "%s\n", string(data))
}

func listJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text;charset=UTF-8")
	currjob := getCurrentJob()
	for _, job := range pendingJobs() {
		if job.BuildId == currjob.BuildId {
			fmt.Fprintf(w, "EXEC: %v\n", job.String())
		} else {
			fmt.Fprintf(w, "PEND: %v\n", job.String())
		}
	}
}

func createBuild(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	buildId := getConfig("build_id").(int) + 1
	setConfig("build_id", buildId)
	vars := mux.Vars(r)
	params := r.URL.Query()
	os := vars["OS"]
	repo := "git://github.com/couchbase/manifest"
	if repos, ok := params["repo"]; ok && len(repos) > 0 {
		repo = repos[0]
	}
	xmlFile := params["xmlfile"][0]
	job := newBuildJob(buildId, os, repo, xmlFile)

	now, done := time.Now(), make(chan bool)
	seconds := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			time.Sleep(time.Second * 1)
			seconds++
			select {
			case <-done:
				fmt.Fprintf(w, "Done\n")
				fmt.Fprintf(w, "Time elapsed: %s\n", time.Since(now))
				flusher.Flush()
				close(done)
				return
			default:
				fmt.Fprintf(w, ".")
				if (seconds % 60) == 0 {
					fmt.Fprintf(w, "\n")
				}
				flusher.Flush()
			}
		}
	}()
	job = job.run(w)
	done <- true
	wg.Wait()

	fmt.Fprintf(w, "S3 download links:\n")
	fmt.Fprintf(w, "  %s\n", job.cbServer)
	fmt.Fprintf(w, "  %s\n", job.cbDebugServer)
}

func createTest(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	buildId := getConfig("build_id").(int) + 1
	setConfig("build_id", buildId)

	// Gather cluster node count, test-case ini and conf files
	params := r.URL.Query()
	repo := "git://github.com/couchbase/manifest"
	if repos, ok := params["repo"]; ok && len(repos) > 0 {
		repo = repos[0]
	}
	xmlFile := params["xmlfile"][0]
	nodeCount, err := strconv.Atoi(params["nodeCount"][0])
	if err != nil {
		fmt.Fprintf(w, "Invalid nodeCount passed\n")
		return
	}
	iniFile := params["ini"][0]
	confFile := params["conf"][0]

	job := newTestRunnerJob(buildId, nodeCount, repo,
		xmlFile, iniFile, confFile)

	now, done := time.Now(), make(chan bool)
	seconds := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			time.Sleep(time.Second * 1)
			seconds++
			select {
			case <-done:
				fmt.Fprintf(w, "Done\n")
				fmt.Fprintf(w, "Time elapsed: %s\n", time.Since(now))
				flusher.Flush()
				close(done)
				return
			default:
				fmt.Fprintf(w, ".")
				if (seconds % 60) == 0 {
					fmt.Fprintf(w, "\n")
				}
				flusher.Flush()
			}
		}
	}()
	job = job.run(w)
	done <- true
	wg.Wait()

	fmt.Fprintf(w, "S3 link for cbcollect_info archive:\n")
	fmt.Fprintf(w, " %s\n", job.cbCollectLog)
}

func argParse() {
	flag.StringVar(&options.basedir, "dir", "/var/tmp",
		"directory path to save builder files (configuration, log)")
	flag.StringVar(&options.logfile, "log", "",
		"directory path to save builder files (configuration, log)")
	if options.logfile != "" {
		logfile := filepath.Join(options.basedir, options.logfile)
		w, err := os.OpenFile(logfile, os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			log.Fatalf("opening file %s: %v", logfile, err)
		}
		log.SetOutput(w)
	}
	flag.Parse()
}

func main() {
	argParse()

	loadConfig(filepath.Join(options.basedir, "builder.json"))
	go runBuildJobs()
	go runTestRunnerJobs()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", welcomeHandler)
	router.HandleFunc("/OS", getOSList)
	router.HandleFunc("/uptime", upTime)
	router.HandleFunc("/config", getConfiguration)
	router.HandleFunc("/listjobs", listJobs)
	router.HandleFunc("/build/{OS}", createBuild)
	router.HandleFunc("/testrunner", createTest)
	fmt.Println("Starting web service on 8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
