package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
)

/**
TODOs
- Read Metadata from environment variables
- Populate the DTO with the metadata
- Dockerize the application
- Make the endpoint configurable
- Make the test results directory configurable
*/

type ResultMetadata struct {
	AssignmentRepoBranchName string `json:"assignmentRepoBranchName"`
	AssignmentRepoCommitHash string `json:"assignmentRepoCommitHash"`
	TestsRepoCommitHash      string `json:"testsRepoCommitHash"`
	IsBuildSuccessful        bool   `json:"isBuildSuccessful"`
	BuildRunDate             string `json:"buildRunDate"`
}
type ResultDTO struct {
	ResultMetadata
	BuildJobs []junit.Suite `json:"buildJobs"`
}

const (
	APIendpoint = "http://localhost:3001/result"
)

func main() {
	log.Info("Starting the application...")

	suites, err := junit.IngestDir("test-data/build/test-results/test")
	if err != nil {
		log.Fatal(err)
	}

	reportResult(suites)
}

func reportResult(suites []junit.Suite) {
	var resultDTO ResultDTO
	for _, suite := range suites {
		resultDTO.BuildJobs = append(resultDTO.BuildJobs, suite)
	}

	// Convert the DTO to JSON
	jsonData, err := json.Marshal(resultDTO)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new request using http
	req, err := http.NewRequest("POST", APIendpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create a Client
	client := &http.Client{}

	// Send the request via a client
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

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
