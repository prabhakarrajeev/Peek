package service

import (
	"flag"
	"fmt"
	"github.com/kavu/go_reuseport"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type Handler func(http.ResponseWriter, *http.Request)

var (
	port *int
)

func Handle(patterns []string, handler Handler) {

	for _, pattern := range patterns {
		http.HandleFunc(pattern, handler)
	}
}

func Init() {
	port = flag.Int("p", 3000, "Listen as a webserver from the specified TCP port number")
	threads := flag.Int("t", int(math.Max(2, float64(runtime.NumCPU())/4)), "Number of active threads")

	flag.Parse()

	// Utilize all CPU cores for serving requests
	runtime.GOMAXPROCS(*threads)
}

func Start() error {
	// Allow a graceful exit
	sigchan := make(chan os.Signal)

	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigchan
		os.Exit(0)
	}()

	// Allow multiple service instances listening on the same port if supported by the OS
	listener, err := reuseport.NewReusablePortListener("tcp4", fmt.Sprintf(":%d", *port))
	if err != nil {
		return http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}

	// Start listening from port and handling requests
	defer listener.Close()
	server := &http.Server{}
	return server.Serve(listener)
}
