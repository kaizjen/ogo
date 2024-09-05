package lib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func stringError(writer http.ResponseWriter, err string) {
	writer.WriteHeader(400)
	writer.Write([]byte("ERROR: " + err))
}

func EnqueueEndpoint(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Access-Control-Allow-Origin", "*")
	//                   ^ Used so the ./index.html works without CORS errors

	amount, err := strconv.Atoi(request.FormValue("n"))
	if err != nil {
		stringError(writer, "Malformed n")
		return
	}

	delta, err := strconv.ParseFloat(request.FormValue("d"), 64)
	if err != nil {
		stringError(writer, "Malformed d")
		return
	}

	start, err := strconv.ParseFloat(request.FormValue("n1"), 64)
	if err != nil {
		stringError(writer, "Malformed n1")
		return
	}

	interval, err := strconv.ParseFloat(request.FormValue("I"), 64)
	if err != nil {
		stringError(writer, "Malformed I")
		return
	}

	ttl, err := strconv.ParseFloat(request.FormValue("TTL"), 64)
	if err != nil {
		stringError(writer, "Malformed TTL")
		return
	}

	task := Task{
		ID:            nextTaskID,
		MaxIterations: amount,
		Delta:         delta,
		CurrentValue:  start,
		StartValue:    start,
		Iteration:     1,
		Interval:      interval,
		TTL:           ttl,
		Status:        STATUS_IN_QUEUE,
		EnqueuedAt:    time.Now().Unix(),
	}

	bytes, err := json.Marshal(task)
	if err != nil {
		stringError(writer, "Whoops")
		fmt.Printf("Error: Invalid (%v)\n", err)
		return
	}
	writer.Write(bytes)

	enqueue(&task)
}

func ListEndpoint(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Access-Control-Allow-Origin", "*")
	//                   ^ Used so the ./index.html works without CORS errors

	taskList := list()

	bytes, err := json.Marshal(taskList)
	if err != nil {
		stringError(writer, "Error: "+err.Error())
		return
	}

	writer.Write(bytes)
}
