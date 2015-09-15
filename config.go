package main

import "io/ioutil"
import "log"
import "encoding/json"
import "sync"

var config Config
var muConfig sync.Mutex

type Config struct {
	filename        string
	BuildId         int `json:"build_id"`
	TotalBuilds     int `json:"total_builds"`
	TotalTests      int `json:"total_tests"`
	CompletedBuilds int `json:"completed_builds"`
	CompletedTests  int `json:"completed_tests"`
}

func loadConfig(configfile string) {
	muConfig.Lock()
	defer muConfig.Unlock()

	data, err := ioutil.ReadFile(configfile)
	if err == nil {
		err := json.Unmarshal(data, &config)
		if err != nil {
			log.Fatalf("error decoding config %v: %v", configfile, err)
		}
	} else {
		config.BuildId = 0
		config.TotalBuilds = 0
		config.TotalTests = 0
		config.CompletedBuilds = 0
		config.CompletedTests = 0
	}
	config.filename = configfile
}

func marshalConfig() ([]byte, error) {
	muConfig.Lock()
	defer muConfig.Unlock()

	return json.Marshal(config)
}

func saveConfig() {
	muConfig.Lock()
	defer muConfig.Unlock()

	data, err := json.Marshal(&config)
	if err != nil {
		log.Fatalf("error decoding config: %v", err)
	}
	err = ioutil.WriteFile(config.filename, data, 0660)
	if err != nil {
		log.Fatalf("error writing config %v: %v", config.filename, err)
	}
}

func setConfig(key string, value interface{}) interface{} {
	muConfig.Lock()
	defer saveConfig()
	defer muConfig.Unlock()

	var old interface{}
	switch key {
	case "build_id":
		old = config.BuildId
		config.BuildId = value.(int)
	case "total_builds":
		old = config.TotalBuilds
		config.TotalBuilds = value.(int)
	case "total_tests":
		old = config.TotalTests
		config.TotalTests = value.(int)
	case "completed_builds":
		old = config.CompletedBuilds
		config.CompletedBuilds = value.(int)
	case "completed_tests":
		old = config.CompletedTests
		config.CompletedTests = value.(int)
	}
	return old
}

func getConfig(key string) interface{} {
	muConfig.Lock()
	defer muConfig.Unlock()

	var value interface{}
	switch key {
	case "build_id":
		value = config.BuildId
	case "total_builds":
		value = config.TotalBuilds
	case "total_tests":
		value = config.TotalTests
	case "completed_builds":
		value = config.CompletedBuilds
	case "completed_tests":
		value = config.CompletedTests
	}
	return value
}
