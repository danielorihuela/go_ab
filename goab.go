package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/danielorihuela/goab/logger"
)

var log = logger.New(false, logger.DebugLevel)

const UnsuccessfulRequest = 0
const SuccessfulRequest = 1

func pageIsNotReachable(resp *http.Response, err error) bool {
	if err != nil || resp.StatusCode == 404 {
		return true
	}
	return false
}

func sendRequest(client *http.Client, testUrl string) int {
	resp, err := client.Get(testUrl)
	if err != nil {
		log.Error(err)
		return UnsuccessfulRequest
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	return SuccessfulRequest
}

func main() {
	numberRequestsPtr := flag.Int("n", 1, "Number of requests to make")
	numberConcurrentConnectionsPtr := flag.Int("c", 1, "Number of concurrent connections")
	keepAlivePtr := flag.Bool("k", false, "Activate keep alive HTTP feature")
	flag.Parse()

	testUrl := os.Args[len(os.Args)-1]

	log.Debug("Url to test =", testUrl)
	log.Debug("Number of requests =", *numberRequestsPtr)
	log.Debug("Concurrent requests =", *numberConcurrentConnectionsPtr)
	log.Debug("Keep Alive HTTP is activated =", *keepAlivePtr)

	resp, _ := http.Get(testUrl)
	fmt.Println(*resp)

	resp, err := http.Get(testUrl)
	if pageIsNotReachable(resp, err) {
		fmt.Println("The introduced url cannot be reached.")
		os.Exit(1)
	}

	requests := make(chan int)
	results := make(chan int, *numberRequestsPtr)

	transport := &http.Transport{DisableKeepAlives: !*keepAlivePtr}
	client := &http.Client{Transport: transport}

	startRequests := time.Now()
	for connectionId := 0; connectionId < *numberConcurrentConnectionsPtr; connectionId++ {
		go func(connectionId int) {
			for range requests {
				results <- sendRequest(client, testUrl)
			}
		}(connectionId)
	}

	for requestId := 0; requestId < *numberRequestsPtr; requestId++ {
		requests <- requestId
	}
	close(requests)

	totalResults := 0
	for resultPosition := 0; resultPosition < *numberRequestsPtr; resultPosition++ {
		result := <-results
		if result == SuccessfulRequest {
			totalResults += 1
		}
	}
	timeTaken := time.Since(startRequests).Seconds()

	fmt.Println("Total time of program in seconds", timeTaken)
	fmt.Println("Requests per second", float64(*numberRequestsPtr)/timeTaken)
	fmt.Println("Time per request (mean)", float64(*numberConcurrentConnectionsPtr)*timeTaken*1000/float64(*numberRequestsPtr))
	fmt.Println("Time per request (mean, across all concurrent requests)", timeTaken*1000/float64(*numberRequestsPtr))

	fmt.Println("Errored responses =", *numberRequestsPtr-totalResults)
	errorPercentage := (1 - (float64(totalResults) / float64(*numberRequestsPtr))) * 100
	fmt.Println("Errored responses percentage =", strconv.FormatFloat(errorPercentage, 'f', 2, 64)+"%")
}
