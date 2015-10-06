package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

var jobs = make(map[int]Job)
var currJob *Job
var muJobs sync.Mutex
var reqBuildch = make(chan Job)
var reqTestRunnerch = make(chan Job)

type Job struct {
	JobType       string `json:"jobType"`
	BuildId       int    `json:"buildId"`
	Os            string `json:"os"`
	Repo          string `json:"repo"`
	BuildXML      string `json:"buildxml"`
	IniFile       string `json:"iniFile"`
	ConfFile      string `json:"confFile"`
	NodeCount     int    `json:"nodeCount"`
	respch        chan Job
	cbServer      string
	cbDebugServer string
	cbCollectLog  string
}

func newBuildJob(buildId int, os, repo, buildXML string) Job {
	job := Job{
		JobType:  "build",
		BuildId:  buildId,
		Os:       os,
		Repo:     repo,
		BuildXML: buildXML,
	}
	return job
}

func newTestRunnerJob(buildId, nodeCount int,
	repo, buildXML, iniFile, confFile string) Job {
	job := Job{
		JobType:   "testrunner",
		BuildId:   buildId,
		Repo:      repo,
		NodeCount: nodeCount,
		BuildXML:  buildXML,
		IniFile:   iniFile,
		ConfFile:  confFile,
	}
	return job
}

func (job Job) manifestPath() string {
	return filepath.Join(job.Repo, job.BuildXML)
}

func (job Job) MarshalJSON() ([]byte, error) {
	return json.Marshal(job)
}

func (job Job) String() string {
	return fmt.Sprintf("(BuildID-%v) %s ; %s", job.BuildId, job.JobType, job.manifestPath())
}

func (job Job) run(w io.Writer) Job {
	job.respch = make(chan Job)
	addJob(job)
	fmt.Fprintf(w, "reqjob: %v\n", job.String())
	if job.JobType == "build" {
		reqBuildch <- job
	} else if job.JobType == "testrunner" {
		reqTestRunnerch <- job
	}
	fmt.Fprintf(w, "runjob: %v\n", job.String())
	return <-job.respch // wait until the job gets picked up.
}

func addJob(job Job) {
	muJobs.Lock()
	defer muJobs.Unlock()
	if _, ok := jobs[job.BuildId]; ok {
		log.Fatalf("job %v already accepted", job.String())
	}
	if job.JobType == "build" {
		total_builds := getConfig("total_builds").(int)
		setConfig("total_builds", total_builds+1)
	} else if job.JobType == "testrunner" {
		total_tests := getConfig("total_tests").(int)
		setConfig("total_tests", total_tests+1)
	}
	log.Printf("addjob: %v\n", job.String())
	jobs[job.BuildId] = job
}

func delJob(job Job) {
	muJobs.Lock()
	defer muJobs.Unlock()
	delete(jobs, job.BuildId)
	log.Printf("deljob: %v\n", job.String())
}

func pendingJobs() []Job {
	muJobs.Lock()
	defer muJobs.Unlock()
	pjobs := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		pjobs = append(pjobs, job)
	}
	return pjobs
}

func setCurrentJob(job *Job) {
	muJobs.Lock()
	defer muJobs.Unlock()
	currJob = job
}

func getCurrentJob() *Job {
	muJobs.Lock()
	defer muJobs.Unlock()
	return currJob
}

func runBuildJobs() {
	for {
		job := <-reqBuildch
		log.Printf("runjob: %v\n", job.String())
		setCurrentJob(&job)
		buildId, os := job.BuildId, job.Os
		repo, xmlFile, uuid := job.Repo, job.BuildXML, jobUUID()
		cmd := exec.Command(
			"./createBuild.sh",
			osBuildMapping[os],
			os,
			repo,
			xmlFile,
			strconv.Itoa(buildId),
			uuid)

		cmdOutput := &bytes.Buffer{}
		cmd.Stdout = cmdOutput
		cmd.Stderr = cmdOutput

		if err := cmd.Start(); err != nil {
			log.Printf("cmd.Start(): %v\n", err)
		}
		cmd.Wait()

		ext := "deb"
		if os == "centos6" || os == "centos7" {
			ext = "rpm"
		}

		// S3 download links:
		if ext == "rpm" {
			job.cbServer = `http://customers.couchbase.com.s3.amazonaws.com` +
				`/couchbase/couchbase-server-1.7~toy-10` +
				strconv.Itoa(buildId) + `.0.0.x86_64.rpm`
			job.cbDebugServer = `http://customers.couchbase.com.s3.amazonaws.com` +
				`/couchbase/couchbase-server-debug-1.7~toy-10` +
				strconv.Itoa(buildId) + `.0.0.x86_64.rpm`
		} else {
			job.cbServer = `http://customers.couchbase.com.s3.amazonaws.com` +
				`/couchbase/couchbase-server-enterprise_toy-10` +
				strconv.Itoa(buildId) + `.0.0-1-debian7_amd64.deb`
			job.cbDebugServer = `http://customers.couchbase.com.s3.amazonaws.com` +
				`/couchbase/couchbase-server-enterprise-dbg_toy-10` +
				strconv.Itoa(buildId) + `.0.0-1-debian7_amd64.deb`

		}
		job.respch <- job
		delJob(job)
		setCurrentJob(nil)
		builds := getConfig("completed_builds").(int)
		setConfig("completed_builds", builds+1)
	}
}

func runTestRunnerJobs() {
	for {
		job := <-reqTestRunnerch
		log.Printf("runjob: %v\n", job.String())
		setCurrentJob(&job)
		repo, xmlFile, nodeCount := job.Repo, job.BuildXML, job.NodeCount
		iniFile, confFile, uuid := job.IniFile, job.ConfFile, jobUUID()

		cmd := exec.Command(
			"./runTests.sh",
			osBuildMapping["centos6"],
			repo,
			xmlFile,
			strconv.Itoa(nodeCount),
			iniFile,
			confFile,
			uuid)

		cmdOutput := &bytes.Buffer{}
		cmd.Stdout = cmdOutput
		cmd.Stderr = cmdOutput

		if err := cmd.Start(); err != nil {
			log.Printf("cmd.Start(): %v\n", err)
		}
		cmd.Wait()

		job.cbCollectLog = `https://s3.amazonaws.com/` +
			`customers.couchbase.com/couchbase/` + uuid + `.zip`

		job.respch <- job
		delJob(job)
		setCurrentJob(nil)
		tests := getConfig("completed_tests").(int)
		setConfig("completed_tests", tests+1)

	}
}

func jobUUID() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		log.Printf("Failed to generate job uuid: %v\n", err)
		return ""
	}
	return hex.EncodeToString(b)
}
