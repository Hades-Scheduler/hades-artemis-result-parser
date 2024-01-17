package main

import (
	"bytes"
	"encoding/json"
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
	JobName                  string `json:"jobName" env:"JOB_NAME"`
	AssignmentRepoBranchName string `json:"assignmentRepoBranchName" env:"ASSIGNMENT_REPO_BRANCH_NAME" envDefault:"main"`
	IsBuildSuccessful        bool   `json:"isBuildSuccessful" env:"IS_BUILD_SUCCESSFUL"`
	//AssignmentRepoCommitHash string `json:"assignmentRepoCommitHash" env:"ASSIGNMENT_REPO_COMMIT_HASH"`
	//TestsRepoCommitHash      string `json:"testsRepoCommitHash" env:"TESTS_REPO_COMMIT_HASH"`
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
	log.Info("Parse JUnit results to DTOs...")
	suites, err := junit.IngestDir(config.IngestDir)
	if err != nil {
		//TODO: Create response with error message
		markBuildAsFailed()
		log.Errorf("Could not Parse JUnit XML files - %v ", err)
	} else {
		markBuildAsSuccessful()
		log.Info("Successfully parsed the JUnit results")
	}

	jsonbody := parseResults(suites)
	sendResponse(jsonbody)
}

func loadEnv() {
	log.Info("Loading Environment variables...")
	// Load the environment variables
	if err := env.Parse(&config); err != nil {
		log.Fatal(err)
	}
	log.Debugf("App Config: %+v", config)

	if err := env.Parse(&metadata); err != nil {
		log.Fatal(err)
	}
	log.Debugf("Metadata Config: %+v", metadata)
	log.Info("Environment variables loaded successfully")
}

// Parses the JUnit results and creates a JSON representation
func parseResults(suites []junit.Suite) []byte {
	log.Info("Parsing the JUnit results to JSON...")
	metadata.BuildJobs = append(metadata.BuildJobs, suites...)

	// Convert the DTO to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("Error parsing JUnit results to JSON %v", err)
	}
	log.Debug("JSON Data: ", string(jsonData))
	return jsonData
}

func markBuildAsFailed() {
	metadata.IsBuildSuccessful = false
	// TODO: Add more details to the response
}

func markBuildAsSuccessful() {
	metadata.IsBuildSuccessful = true
	// TODO: Add more details to the response
}

func sendResponse(json []byte) {
	// Create a new request using http
	log.Info("Sending the response to the API...")
	req, err := http.NewRequest("POST", config.APIendpoint, bytes.NewBuffer(json))
	if err != nil {
		log.Debug("Error creating the request")
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
		log.Debug("Error sending the request")
		log.Fatal(err)
	}

	log.Info("Response Status: ", resp.Status)

	// Close response body
	defer resp.Body.Close()
}
