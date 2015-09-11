package main

import "bytes"
import "encoding/json"
import "fmt"
import "io"
import "log"
import "os/exec"
import "strconv"
import "time"
import "sync"
import "path/filepath"

var jobs = make(map[int]Job)
var currJob *Job
var muJobs sync.Mutex
var reqch = make(chan Job)

type Job struct {
	BuildId       int    `json:"buildId"`
	Os            string `json:"os"`
	Repo          string `json:"repo"`
	BuildXML      string `json:"buildxml"`
	respch        chan Job
	cbServer      string
	cbDebugServer string
}

func newJob(buildId int, os, repo, buildXML string) Job {
	job := Job{BuildId: buildId, Os: os, Repo: repo, BuildXML: buildXML}
	return job
}

func (job Job) manifestPath() string {
	return filepath.Join(job.Repo, job.BuildXML)
}

func (job Job) MarshalJSON() ([]byte, error) {
	return json.Marshal(job)
}

func (job Job) String() string {
	return fmt.Sprintf("(BuildID-%v) %s ; %s", job.BuildId, job.Os, job.manifestPath())
}

func (job Job) run(w io.Writer) Job {
	job.respch = make(chan Job)
	addJob(job)
	reqch <- job
	fmt.Fprintf(w, "runjob: %v\n", job.String())
	return <-job.respch // wait until the job gets picked up.
}

func addJob(job Job) {
	muJobs.Lock()
	defer muJobs.Unlock()
	if _, ok := jobs[job.BuildId]; ok {
		log.Fatalf("job %v already accepted", job.String())
	}
	total_builds := getConfig("total_builds").(int)
	setConfig("total_builds", total_builds+1)
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

func runJobs() {
	for {
		job := <-reqch
		log.Printf("runjob: %v\n", job.String())
		setCurrentJob(&job)
		buildId, os := job.BuildId, job.Os
		repo, xmlFile := job.Repo, job.BuildXML
		cmd := exec.Command(
			"./createBuild.sh", osBuildMapping[os],
			os, repo, xmlFile, strconv.Itoa(buildId))

		cmdOutput := &bytes.Buffer{}
		cmd.Stdout = cmdOutput
		cmd.Stderr = cmdOutput

		if err := cmd.Start(); err != nil {
			log.Printf("cmd.Start(): %v\n", err)
		}
		cmd.Wait()
		log.Printf("%v\n", string(cmdOutput.Bytes()))

		// Sleep to allow dump of timing stats
		time.Sleep(2 * time.Second)

		ext := "dep"
		if os == "centos6" || os == "centos7" {
			ext = "rpm"
		}

		// S3 download links:
		job.cbServer = `http://customers.couchbase.com.s3.amazonaws.com` +
			`/couchbase/couchbase-server-1.7~toy-10` +
			strconv.Itoa(buildId) + `.0.0.0.x86_64.` + ext
		job.cbDebugServer = `http://customers.couchbase.com.s3.amazonaws.com` +
			`/couchbase/couchbase-server-debug-1.7~toy-10` +
			strconv.Itoa(buildId) + `.0.0.0.x86_64.` + ext
		job.respch <- job
		delJob(job)
		setCurrentJob(nil)
		builds := getConfig("completed_builds").(int)
		setConfig("completed_builds", builds+1)
	}
}
