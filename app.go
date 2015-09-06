package main

import (
    "encoding/json"
    "os/exec"
    "fmt"
    "log"
    "net/http"

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

    js, err := json.Marshal(OSList);
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
    xmlFile := vars["xmlFile"]

    if err := exec.Command("./createBuild.sh",
            osBuildMapping[os], os, xmlFile).Run(); err != nil {
                log.Println(err)
            }
    log.Printf("Successfully create build for os: %s xml: %s", os, xmlFile)

    fmt.Fprintf(w, "Started build for OS: %v xmlFile: %v", os, xmlFile)
}

func main() {
    //Construct OS names to docker build mapping
    osPortMapping()

    router := mux.NewRouter().StrictSlash(true)

    router.HandleFunc("/", handler)
    router.HandleFunc("/OS", getOSList)
    router.HandleFunc("/build/{OS}/{xmlFile}", createBuild)
    fmt.Println("Starting web service on 10000")
    http.ListenAndServe(":10000", router)
}
