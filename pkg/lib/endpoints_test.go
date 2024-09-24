package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func testListEndpoint(t *testing.T, exec *Executor) []Task {
	req, err := http.NewRequest("GET", "/tasks", nil)

	if err != nil {
		FailTest(t, "Failed to create a request")
	}

	rec := httptest.NewRecorder()
	exec.ListEndpoint(rec, req)

	if rec.Result().StatusCode != 200 {
		FailTest(t, "invalid status code (wanted 200, got %v)", rec.Result().StatusCode)
	}

	tasks := []Task{}

	jsonErr := json.Unmarshal(rec.Body.Bytes(), &tasks)
	if jsonErr != nil {
		FailTest(t, "can't unmarshal list: %v", jsonErr)
	}

	return tasks
}

func makeEnqueueParameters(n, d, n1, i, ttl string) string {
	return fmt.Sprintf(`{ "n": %v, "d": %v, "n1": %v, "I": %v, "TTL": %v }`, n, d, n1, i, ttl)
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

func makeEnqueueRequest(t *testing.T, exec *Executor, params string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rec.Body.Write([]byte(params))

	req, err := http.NewRequest("POST", "/tasks", rec.Body)

	if err != nil {
		FailTest(t, "Failed to create a request")
	}

	exec.EnqueueEndpoint(rec, req)

	return rec
}

// This function sets a bunch of properties
// on its tasks - that's why it copies them
// (just in case it's important that the task is preserved)
func compareTasks(t1, t2 Task) bool {
	// Ignore the time, because it's too complicated
	// to compare - it might be off just a little bit
	// and it's just a call to time.Now().Unix() so
	t1.EnqueuedAt = 0
	t2.EnqueuedAt = 0
	t1.StartedAt = 0
	t2.StartedAt = 0
	t1.CompletedAt = 0
	t2.CompletedAt = 0
	return t1 == t2
}

func testHTTPEnqueue(t *testing.T, exec *Executor, n int, d, n1, i, ttl float64) {
	rec := makeEnqueueRequest(t, exec, makeEnqueueParameters(
		intToString(n), floatToString(d), floatToString(n1),
		floatToString(i), floatToString(ttl),
	))

	if rec.Result().StatusCode != 200 {
		FailTest(t, "invalid status code (wanted 200, got %v)", rec.Result().StatusCode)
	}

	wantResponseBytes, marshalError := json.Marshal(map[string]any{
		"success": true,
	})
	if marshalError != nil {
		FailTest(t, "cannot marshal: %v", marshalError)
	}
	response := rec.Body.String()
	if string(wantResponseBytes) != response {
		FailTest(t, "incorrect response (want %v, got %v)", string(wantResponseBytes), response)
	}
}

func TestSingleEnqueue(t *testing.T) {
	t.Logf("TestSingleEnqueue:")
	exec := NewExecutor()

	n, interval := 3, 1.4

	// Use float values whenever possible in case something breaks
	// because of floating point impersicion
	testHTTPEnqueue(t, &exec, n, 1.5, 2.2, interval, 2.3)
	// testHTTPEnqueue returns us a "fake" task, sent by the endpoint
	// here's how we get the real one:
	task := exec.currentlyExecuting[0]

	list := testListEndpoint(t, &exec)
	if len(list) < 1 || !compareTasks(*task, *&list[0]) {
		FailTest(t, "invalid list of tasks: wanted [%v], got %v", task, list)
	}

	time.Sleep(floatSecondsToDuration(interval * float64(n)))

	testTaskCompletion(t, task)
	waitForTaskDestruction(t, task)
	testTaskDestruction(t, &exec, task)

	list = testListEndpoint(t, &exec)
	if len(list) != 0 {
		FailTest(t, "length of list is non-zero")
	}
}

func TestMalformed(t *testing.T) {
	exec := (NewExecutor())

	reqs := []*httptest.ResponseRecorder{
		makeEnqueueRequest(t, &exec, makeEnqueueParameters(`-invalid-json-`, "0", "0", "0", "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters(`"abc"`, "0", "0", "0", "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters("2.3", "0", "0", "0", "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters("2", `"def"`, "0", "0", "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters("2", "5", `"ghi"`, "0", "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters("2", "5", "45", `"jkl"`, "0")),
		makeEnqueueRequest(t, &exec, makeEnqueueParameters("2", "5", "45", "7", `"x"`)),
	}

	wantBodyBytes, marshalError := json.Marshal(map[string]any{
		"success": false,
		"error":   "Malformed request",
	})
	if marshalError != nil {
		FailTest(t, "cannot marshal `wantBody`")
	}

	wantBody := string(wantBodyBytes)

	for _, req := range reqs {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			FailTest(t, "cannot read response body")
		}
		if string(body) != wantBody {
			FailTest(t, "unexpected body (want %s, got %s)", body, wantBody)
		}
		if req.Result().StatusCode != 400 {
			FailTest(t, "expected HTTP code 400")
		}
	}
}
