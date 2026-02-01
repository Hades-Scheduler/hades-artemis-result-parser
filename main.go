package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v9"
	git "github.com/go-git/go-git/v5"
	"github.com/joshdk/go-junit"
	"github.com/ls1intum/hades/shared/buildlogs"
	log "github.com/sirupsen/logrus"
)

// App config populated from environment variables
type Config struct {
	APIendpoint        string `env:"API_ENDPOINT"`
	APIToken           string `env:"API_TOKEN"`
	IngestDir          string `env:"INGEST_DIR"`
	TestRepoPath       string `env:"HADES_TEST_PATH"`
	AssignmentRepoPath string `env:"HADES_ASSIGNMENT_PATH"`
}

// ResultMetadata populated from environment variables
type ResultMetadata struct {
	JobName                  string `json:"jobName" env:"JOB_NAME"`
	UUID                     string `json:"uuid" env:"UUID"`
	AssignmentRepoBranchName string `json:"assignmentRepoBranchName" env:"ASSIGNMENT_REPO_BRANCH_NAME" envDefault:"main"`
	IsBuildSuccessful        bool   `json:"isBuildSuccessful" env:"IS_BUILD_SUCCESSFUL"`
	AssignmentRepoCommitHash string `json:"assignmentRepoCommitHash" env:"ASSIGNMENT_REPO_COMMIT_HASH"`
	TestsRepoCommitHash      string `json:"testsRepoCommitHash" env:"TESTS_REPO_COMMIT_HASH"`
	BuildCompletionTime      string `json:"buildCompletionTime" env:"BUILD_COMPLETION_TIME"`
}
type ResultDTO struct {
	ResultMetadata
	BuildJobs []junit.Suite `json:"buildJobs"`
	BuildLogs buildlogs.Log `json:"buildLogs"`
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

	metadata.AssignmentRepoCommitHash = getCommitHash(config.AssignmentRepoPath)
	metadata.TestsRepoCommitHash = getCommitHash(config.TestRepoPath)
	metadata.BuildCompletionTime = time.Now().Format(time.RFC3339)

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
func getCommitHash(repoPath string) string {
	log.Debug("Getting the commit hash for path: ", repoPath)
	// Open the repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		log.WithError(err).Warn("Could not open the repository")
		return ""
	}
	log.Debug("Successfully opened the repository")

	// Get the HEAD reference
	ref, err := r.Head()
	if err != nil {
		log.Warn("Commit hash not fount - ", err)
	}
	log.Infof("Commit Hash for path %s is: %s ", repoPath, ref.Hash().String())
	// Return the commit hash
	return ref.Hash().String()
}

func markBuildAsFailed() {
	metadata.IsBuildSuccessful = false
	log.Info("Build failed")
	// TODO: Add more details to the response
}

func markBuildAsSuccessful() {
	metadata.IsBuildSuccessful = true
	log.Info("Build successful")
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
