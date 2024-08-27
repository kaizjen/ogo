// This is kind of annoying because i wanted to name
// my whole module "ogo" so i could have "ogo.go" file names.
// but you apparently can't run anything that isn't named "main" :(
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"
)

var N = 10

const ADDR = ":3000"

type Status int

const (
	STATUS_IN_QUEUE    Status = iota
	STATUS_IN_PROGRESS Status = iota
	STSTUS_COMPLETED   Status = iota
)

type Task struct {
	ID     int
	Status Status
	// n
	MaxIterations int
	// d
	Delta        float64
	CurrentValue float64
	StartValue   float64
	Iteration    int
	// I (seconds)
	Interval int
	// TTL (seconds)
	TTL int

	// (unix seconds)
	EnqueuedAt int64
	// (unix seconds)
	StartedAt int64
	// (unix seconds)
	CompletedAt int64
}

// Locking queue, currentlyExecuting
var mutex = sync.Mutex{}

// Using mutexes seems way simpler than using a sync channel
// though sync channels may be more robust/performant, idk

var queue = []*Task{}

//            ^ This is a pointer so that the /list endpoint would be simpler,
//              though i have no idea what memory is allocated on the stack\
//              and what memory is allocated on the heap in go (and ideally
//              we'd like to avoid heap allocs as much as possible)

// Also serves as the total amount of tasks
var nextTaskID = 0
var currentlyExecuting = map[int]*Task{}

// Locking completed
var completedMutex = sync.Mutex{}
var completed = map[int]*Task{}

func stringError(writer http.ResponseWriter, err string) {
	writer.Write([]byte("ERROR: " + err))
}

func execute(task *Task) {
	fmt.Printf("Executing task (id %v)\n", task.ID)

	currentlyExecuting[task.ID] = task

	task.Status = STATUS_IN_PROGRESS
	task.StartedAt = time.Now().Unix()

	for i := 1; i < task.MaxIterations; i++ {
		task.CurrentValue = task.CurrentValue + task.Delta
		task.Iteration++
		// After the last iteration, this sleeps for the Interval
		// not sure if that's the desired behavior
		// (if not, just use an `if`)
		time.Sleep(time.Duration(task.Interval) * time.Second)
	}

	mutex.Lock()
	defer mutex.Unlock()

	fmt.Printf("Task done! (id %v)\n", task.ID)
	delete(currentlyExecuting, task.ID)
	go ttl(task)

	if len(currentlyExecuting) < N {
		if len(queue) > 0 {
			qtask := queue[0]
			queue = queue[1:]
			go execute(qtask)
		}
	}
}

func ttl(task *Task) {
	completedMutex.Lock()
	completed[task.ID] = task
	completedMutex.Unlock()

	task.Status = STSTUS_COMPLETED
	task.CompletedAt = time.Now().Unix()

	time.Sleep(time.Duration(task.TTL) * time.Second)

	completedMutex.Lock()
	fmt.Printf("Task expired! (id %v)\n", task.ID)
	delete(completed, task.ID)
	completedMutex.Unlock()
}

func enqueueEndpoint(writer http.ResponseWriter, request *http.Request) {
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

	interval, err := strconv.Atoi(request.FormValue("I"))
	if err != nil {
		stringError(writer, "Malformed I")
		return
	}

	ttl, err := strconv.Atoi(request.FormValue("TTL"))
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

	mutex.Lock()
	defer mutex.Unlock()

	nextTaskID++
	if len(currentlyExecuting) >= N {
		queue = append(queue, &task)

	} else {
		go execute(&task)
	}
}

func listEndpoint(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Access-Control-Allow-Origin", "*")
	//                   ^ Used so the ./index.html works without CORS errors

	list := make([]*Task, 0, len(completed)+len(currentlyExecuting)+len(queue))

	completedMutex.Lock()
	for _, value := range completed {
		list = append(list, value)
	}
	completedMutex.Unlock()

	mutex.Lock()
	for _, value := range currentlyExecuting {
		list = append(list, value)
	}
	list = append(list, queue...)
	mutex.Unlock()

	slices.SortFunc(list, func(t1, t2 *Task) int {
		if t1.Status == t2.Status {
			return t1.ID - t2.ID
		} else {
			// Sorts tasks like Completed > In progress > In queue
			return int(t2.Status) - int(t1.Status)
		}
	})

	bytes, err := json.Marshal(list)
	if err != nil {
		stringError(writer, "Error: "+err.Error())
		return
	}

	writer.Write(bytes)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-N" {
		if len(os.Args) < 3 {
			fmt.Printf("Error: You must pass a value of N after the argument\n")
			os.Exit(1)
		}
		res, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Error: Invalid N (%s) - it must be an integer\n", os.Args[2])
			os.Exit(1)
		}
		N = res

	} else {
		fmt.Printf("Use `-N <int>` to specify the maximum amount of co-running progressions. Using default value of %d\n", N)
	}

	http.HandleFunc("/enqueue", enqueueEndpoint)
	http.HandleFunc("/list", listEndpoint)

	fmt.Printf("Listening on %v\n", ADDR)
	err := http.ListenAndServe(ADDR, nil)

	if err != nil {
		log.Fatal("Failed to launch the server: ", err)
	}
}
