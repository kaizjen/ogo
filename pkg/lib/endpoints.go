package lib

import (
	"encoding/json"
	"io"
	"net/http"
)

func stringError(writer http.ResponseWriter, err string) {
	writer.WriteHeader(400)
	bytes, marshalError := json.Marshal(map[string]any{
		"success": false,
		"error":   err,
	})
	if marshalError != nil {
		panic("Marshal failed: " + marshalError.Error())
	}
	writer.Write(bytes)
}

func (self *Executor) EnqueueEndpoint(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Access-Control-Allow-Origin", "*")
	//                   ^ Used so the ./index.html works without CORS errors

	buffer, err := io.ReadAll(request.Body)
	if err != nil {
		stringError(writer, "Invalid request")
		return
	}

	parameters := TaskParams{}
	unmarshalErr := json.Unmarshal(buffer, &parameters)
	if unmarshalErr != nil {
		stringError(writer, "Malformed request")
		return
	}

	/* amount, err := strconv.Atoi(request.FormValue("n"))
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
	} */

	writer.Write([]byte(`{"success":true}`))

	self.enqueue(&parameters)
}

func (self *Executor) ListEndpoint(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Access-Control-Allow-Origin", "*")
	//                   ^ Used so the ./index.html works without CORS errors

	taskList := self.list()

	bytes, err := json.Marshal(taskList)
	if err != nil {
		stringError(writer, "Error: "+err.Error())
		return
	}

	writer.Write(bytes)
}
