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
	Passed                   int    `json:"passed" env:"PASSED"`
}

type TestSuiteDTO struct {
	Name      string        `json:"name"`
	Time      float64       `json:"time"`
	Errors    int           `json:"errors"`
	Skipped   int           `json:"skipped"`
	Failures  int           `json:"failures"`
	Tests     int           `json:"tests"`
	TestCases []TestCaseDTO `json:"testCases"`
}

type TestCaseDTO struct {
	Name      string                     `json:"name"`
	Classname string                     `json:"classname"`
	Time      float64                    `json:"time"`
	Failures  []TestCaseDetailMessageDTO `json:"failures"`     // empty for passing tests
	Errors    []TestCaseDetailMessageDTO `json:"errors"`       // empty for passing tests
	Successes []TestCaseDetailMessageDTO `json:"successInfos"` // empty for failing tests
}

type TestCaseDetailMessageDTO struct {
	Message               string `json:"message"`
	Type                  string `json:"type"`
	MessageWithStackTrace string `json:"messageWithStackTrace"`
}

type ResultDTO struct {
	ResultMetadata
	Results   []TestSuiteDTO       `json:"results"`
	BuildLogs []buildlogs.LogEntry `json:"logs"`
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
		log.Errorf("Could not parse JUnit XML files into DTOs- %v ", err)
	} else {
		markBuildAsSuccessful()
		log.Info("Successfully parsed JUnit results into DTOs")
	}

	// Trust the commit hashes provided via environment variables (set by the CI orchestrator,
	// e.g. Artemis). Only fall back to inspecting the cloned working tree when nothing was
	// supplied, since `git rev-parse HEAD` is racy against subsequent pushes and produces the
	// branch name instead of a SHA when the clone container left HEAD on a symbolic ref.
	if metadata.AssignmentRepoCommitHash == "" {
		metadata.AssignmentRepoCommitHash = getCommitHash(config.AssignmentRepoPath)
	}
	if metadata.TestsRepoCommitHash == "" {
		metadata.TestsRepoCommitHash = getCommitHash(config.TestRepoPath)
	}
	metadata.BuildCompletionTime = time.Now().Format(time.RFC3339)
	metadata.BuildLogs = []buildlogs.LogEntry{}

	metadata.Passed = 0
	for i := range suites {
		suites[i].Aggregate()
		metadata.Passed += suites[i].Totals.Passed
	}

	jsonbody := convertResultsToTestSuiteDTO(suites)
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
func parseDTOToJSON(metadata ResultDTO) []byte {
	log.Info("Parsing the JUnit results to JSON...")

	// Convert the DTO to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("Error parsing JUnit results to JSON %v", err)
	}
	log.Debug("JSON Data: ", string(jsonData))
	return jsonData
}

func convertResultsToTestSuiteDTO(suites []junit.Suite) []byte {
	log.Info("Converting JUnit results to TestSuiteDTOs...")

	for _, suite := range suites {
		testCases := make([]TestCaseDTO, len(suite.Tests))
		for i, test := range suite.Tests {
			testCases[i] = TestCaseDTO{
				Name:      test.Name,
				Classname: test.Classname,
				Time:      test.Duration.Seconds(),
			}

			switch test.Status {
			case junit.StatusFailed:
				testCases[i].Failures = []TestCaseDetailMessageDTO{{test.Message, "", test.Error.Error()}}
			case junit.StatusError:
				testCases[i].Errors = []TestCaseDetailMessageDTO{{test.Message, "", test.Error.Error()}}
			default:
				testCases[i].Successes = []TestCaseDetailMessageDTO{{test.Message, "", ""}}
			}
		}

		metadata.Results = append(metadata.Results, TestSuiteDTO{
			Name:      suite.Name,
			Time:      suite.Totals.Duration.Seconds(),
			Errors:    suite.Totals.Error,
			Skipped:   suite.Totals.Skipped,
			Failures:  suite.Totals.Failed,
			Tests:     suite.Totals.Tests,
			TestCases: testCases,
		})
	}

	return parseDTOToJSON(metadata)
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
