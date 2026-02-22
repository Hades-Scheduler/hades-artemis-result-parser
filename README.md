# junit-result-parser

A containerized service that parses generated JUnit XML result files into Artemis-compatible result DTOs and submits them to the Artemis API. It runs as the final step in a Hades CI pipeline after the build and test step has completed.

## Configuration

| Variable | Description |
|---|---|
| `API_ENDPOINT` | Full URL of the Artemis result ingestion endpoint |
| `API_TOKEN` | Bearer token used to authenticate with the Artemis API |
| `INGEST_DIR` | Path inside the container to the directory containing JUnit XML files |
| `JOB_NAME` | Identifier for the CI job (e.g. `EXERCISESHORTNAME-SOLUTION`) |

## Usage

The parser is invoked as a step in a [Hades](https://github.com/ls1intum/hades) job. Environment variables are passed via the step's `metadata` field, and the shared volume from the build step is mounted to provide access to the generated XML files.

Example:
```json
{
  "id": 3,
  "name": "Parse Results",
  "image": "ghcr.io/ls1intum/hades/junit-result-parser:latest",
  "volumeMounts": [
    { "name": "shared", "mountPath": "/shared" }
  ],
  "metadata": {
    "API_ENDPOINT": "http://<artemis-host>/api/public/programming-exercises/new-result",
    "API_TOKEN": "<your-token>",
    "INGEST_DIR": "/shared/example/build/test-results/test",
    "JOB_NAME": "<exercise-job-name>",
    "HADES_TEST_PATH": "/shared/example",
    "HADES_ASSIGNMENT_PATH": "/shared/example/assignment"
  }
}
```
