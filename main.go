// This is kind of annoying because i wanted to name
// my whole module "ogo" so i could have "ogo.go" file names.
// but you apparently can't run anything that isn't named "main" :(
package main

import (
	"fmt"
	"github.com/kaizjen/ogo-disconnected/pkg/lib"
	"net/http"
	"os"
	"strconv"
)

const ADDR = ":3000"

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
		lib.N = res

	} else {
		fmt.Printf("Use `-N <int>` to specify the maximum amount of co-running progressions. Using default value of %d\n", lib.N)
	}

	http.HandleFunc("/enqueue", lib.EnqueueEndpoint)
	http.HandleFunc("/list", lib.ListEndpoint)

	fmt.Printf("Listening on %v\n", ADDR)
	err := http.ListenAndServe(ADDR, nil)

	if err != nil {
		fmt.Printf("Failed to launch the server: %v\n", err)
		os.Exit(1)
	}
}
