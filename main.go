package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Starting the application...")

	suites, err := junit.IngestDir("test-data/build/test-results/test")
	if err != nil {
		log.Fatal(err)
	}
	toJSON(suites)
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
