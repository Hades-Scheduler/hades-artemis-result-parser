package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/caarlos0/env/v9"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
)

// App config populated from environment variables
type Config struct {
	APIendpoint string `env:"API_ENDPOINT"`
	APIToken    string `env:"API_TOKEN"`
	IngestDir   string `env:"INGEST_DIR"`
}

// ResultMetadata populated from environment variables
type ResultMetadata struct {
	JobName                  string `json:"job_name" env:"JOB_NAME"`
	AssignmentRepoBranchName string `json:"assignmentRepoBranchName" env:"ASSIGNMENT_REPO_BRANCH_NAME" envDefault:"main"`
	//AssignmentRepoCommitHash string `json:"assignmentRepoCommitHash" env:"ASSIGNMENT_REPO_COMMIT_HASH"`
	//TestsRepoCommitHash      string `json:"testsRepoCommitHash" env:"TESTS_REPO_COMMIT_HASH"`
	//IsBuildSuccessful        bool   `json:"isBuildSuccessful" env:"IS_BUILD_SUCCESSFUL"`
	//BuildRunDate             string `json:"buildRunDate" env:"BUILD_RUN_DATE"`
}
type ResultDTO struct {
	ResultMetadata
	BuildJobs []junit.Suite `json:"buildJobs"`
}

var config Config
var metadata ResultDTO

func main() {
	log.Info("Starting the application...")
	if is_debug := os.Getenv("DEBUG"); is_debug == "true" {
		log.SetLevel(log.DebugLevel)
		log.Warn("DEBUG MODE ENABLED")
	}
	loadEnv()

	suites, err := junit.IngestDir(config.IngestDir)

	if err != nil {
		log.Fatal(err)
	}

	reportResult(suites)
}

func loadEnv() {
	// Load the environment variables
	if err := env.Parse(&config); err != nil {
		log.Fatal(err)
	}
	log.Debugf("App Config: %+v", config)

	if err := env.Parse(&metadata); err != nil {
		log.Fatal(err)
	}
	log.Debugf("Metadata Config: %+v", metadata)
}

func reportResult(suites []junit.Suite) {
	metadata.BuildJobs = append(metadata.BuildJobs, suites...)

	// Convert the DTO to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("JSON Data: ", string(jsonData))

	// Create a new request using http
	req, err := http.NewRequest("POST", config.APIendpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.APIToken)

	// Create a Client
	client := &http.Client{}

	// Send the request via a client
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Response Status: ", resp.Status)

	// Close response body
	defer resp.Body.Close()
}

func toJSON(suites []junit.Suite) {
	j, err := json.Marshal(suites)
	if err != nil {
		log.Fatal(err)
	}
	var out bytes.Buffer
	json.Indent(&out, j, "", "  ")
	fmt.Print(out.String())

}

func printOutput(suites []junit.Suite) {

	for _, suite := range suites {
		fmt.Println(suite.Name)
		for _, test := range suite.Tests {
			fmt.Printf("  %s\n", test.Name)
			if test.Error != nil {
				fmt.Printf("    %s: %v\n", test.Status, test.Error)
			} else {
				fmt.Printf("    %s\n", test.Status)
			}
		}
	}

}
