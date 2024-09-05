package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func testListEndpoint(t *testing.T) []Task {
	req, err := http.NewRequest("GET", "/list", nil)

	if err != nil {
		FailTest(t, "Failed to create a request")
	}

	rec := httptest.NewRecorder()
	ListEndpoint(rec, req)

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
	return fmt.Sprintf("/enqueue?n=%v&d=%v&n1=%v&I=%v&TTL=%v", n, d, n1, i, ttl)
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

func makeEnqueueRequest(t *testing.T, params string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", params, nil)

	if err != nil {
		FailTest(t, "Failed to create a request")
	}

	rec := httptest.NewRecorder()
	EnqueueEndpoint(rec, req)

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

func testHTTPEnqueue(t *testing.T, n int, d, n1, i, ttl float64) Task {
	rec := makeEnqueueRequest(t, makeEnqueueParameters(
		intToString(n), floatToString(d), floatToString(n1),
		floatToString(i), floatToString(ttl),
	))

	if rec.Result().StatusCode != 200 {
		FailTest(t, "invalid status code (wanted 200, got %v)", rec.Result().StatusCode)
	}

	wantTask := Task{
		ID:            nextTaskID - 1,
		MaxIterations: n,
		Delta:         d,
		CurrentValue:  n1,
		StartValue:    n1,
		Iteration:     1,
		Interval:      i,
		TTL:           ttl,
		Status:        STATUS_IN_QUEUE,
		EnqueuedAt:    -1,
	}
	task := Task{}

	jsonErr := json.Unmarshal(rec.Body.Bytes(), &task)
	if jsonErr != nil {
		FailTest(t, "invalid json sent: %v", jsonErr)
	}

	if !compareTasks(task, wantTask) {
		FailTest(t, "unexpected task (want %v, got %v)", wantTask, task)
	}

	return wantTask
}

func TestSingleEnqueue(t *testing.T) {
	t.Logf("TestSingleEnqueue:")
	// Use float values whenever possible in case something breaks
	// because of floating point impersicion
	testHTTPEnqueue(t, 3, 1.5, 2.2, 1.4, 2.3)
	// testHTTPEnqueue returns us a "fake" task, sent by the endpoint
	// here's how we get the real one:
	task := currentlyExecuting[0]

	list := testListEndpoint(t)
	if len(list) < 1 || !compareTasks(*task, *&list[0]) {
		FailTest(t, "invalid list of tasks: wanted [%v], got %v", task, list)
	}

	waitForTask(t, task)
	testTaskCompletion(t, task)
	waitForTaskDestruction(t, task)
	testTaskDestruction(t, task)

	list = testListEndpoint(t)
	if len(list) != 0 {
		FailTest(t, "length of list is non-zero")
	}
}

type ExpectRequest struct {
	rec  *httptest.ResponseRecorder
	body string
}

func TestMalformed(t *testing.T) {
	reqs := []ExpectRequest{
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("abc", "", "", "", "")),
			body: "ERROR: Malformed n",
		},
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("2.3", "", "", "", "")),
			body: "ERROR: Malformed n",
		},
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("2", "def", "", "", "")),
			body: "ERROR: Malformed d",
		},
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("2", "5", "ghi", "", "")),
			body: "ERROR: Malformed n1",
		},
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("2", "5", "45", "jkl", "")),
			body: "ERROR: Malformed I",
		},
		{
			rec:  makeEnqueueRequest(t, makeEnqueueParameters("2", "5", "45", "7", "x")),
			body: "ERROR: Malformed TTL",
		},
	}

	for _, req := range reqs {
		body, err := io.ReadAll(req.rec.Body)
		if err != nil {
			FailTest(t, "cannot read response body")
		}
		if string(body) != req.body {
			FailTest(t, "unexpected body (want %s, got %s)", req.body, body)
		}
		if req.rec.Result().StatusCode != 400 {
			FailTest(t, "expected HTTP code 400")
		}
	}
}
