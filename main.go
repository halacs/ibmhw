// Requirement list (should not be part of the code ideally)
//
// RQ1: You have to implement a small web service, which can store one user-provided
// unix timestamp (*time.Time) in memory. --- OK
//
// RQ2: The service must have two endpoints, one for saving the timestamp and an other to fetch it. --- OK
//
// RQ3: The only allowed content type on the service side is text/plain for both in and
// egress communications. --- OK
//
// RQ4: The service must take care of data races (concurrent read-write requests on the
// timestamp), but mutexes are not allowed. You should find another way to
// manage the concurrent events. --- OK
//
// RQ5: In the same process where the service is running, please implement the client
// side which first stores a timestamp and then reads it back. --- OK
//
// RQ6: The only output of the application on the standard out (in normal cases) must be
// the timestamp which it has read in the second step. --- OK
//
// RQ7: The output of the exercise has to be two source files (main.go and
// main_test.go). The result must run by executing go run main.go command. --- Ok
//
// RQ8: Test coverage needs to reach at least 2%, maximum allowed coverage is 100%. --- OK 75.9%

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

// Internal data representation of data need to be stored.
type Timestamp struct {
	// Use *time.Time type required by RQ1
	timestamp *time.Time
}

// External data reprensentation: type used in REST communication.
type TimestampJson struct {
	// RQ1 requested to send unix timestamp
	Timestamp int64 `json:"timestamp"`
}

// Response is use to sent simple text message back to the caller of REST endpoint if nothing else would be sent.
type Response struct {
	Response string `json:"response"`
}

// Initial value can be anything, but data pointer must not be nil.
// Fixed value is better for testing.
const initialUnixTimestamp int64 = 1622366082

var now time.Time = time.Unix(initialUnixTimestamp, 0)
var data Timestamp = Timestamp{timestamp: &now}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	log.Debugf("my MarshalJSON called")
	return json.Marshal(TimestampJson{
		// Return unix timestamp required by RQ1
		Timestamp: t.timestamp.Unix(),
	})
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var raw TimestampJson
	log.Debugf("my UnmarshalJSON called. t = %v", t)

	err := json.Unmarshal(b, &raw)
	if err != nil {

		return err
	}

	tmp := time.Unix(raw.Timestamp, 0)
	t.timestamp = &tmp
	log.Debugf("t = %v, t.timestamp = %v", t, t.timestamp)

	return nil
}

// Use this channel for mutual exclusion.
// Only one thread can be in the critical section.
var l = make(chan int, 1) // RQ4

func getData() Timestamp { // RQ4
	l <- 1
	defer func() { <-l }()

	return data
}

func setData(newValue Timestamp) { // RQ4
	l <- 1
	defer func() { <-l }()

	data = newValue
}

func returnTimestamp(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(getData())
}

func storeTimestamp(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	log.Debugf("reqBody: %+v\n", string(reqBody))

	// Parse and store request data
	var newData Timestamp
	if err := json.Unmarshal(reqBody, &newData); err != nil {
		log.Errorf("Unable to parse POST payload as a valid json value. Error: %v", err)

		// Do not expose too much about the internals.
		errorMessage, _ := json.Marshal(Response{"Unable to parse POST payload as a valid json value."})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)

		return
	}
	setData(newData)
	log.Debugf("%v", newData)

	_ = json.NewEncoder(w).Encode(Response{"OK"})
}

func handleRequests(c chan int) {
	log.Debug("Start web server")

	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.HandleFunc("/timestamp", storeTimestamp).Methods("POST").Headers("Content-Type", "text/plain") // RQ2, RQ3
	myRouter.HandleFunc("/timestamp", returnTimestamp).Methods("GET").Headers("Content-Type", "text/plain") // RQ2, RQ3

	// Listen on port 10 000 on all IP addresses of the machine. Log if something bad happen.
	log.Error(http.ListenAndServe(":10000", myRouter))

	log.Debug("Web server exited.")
	c <- 0
}

// Call REST endpoint to set current timestamp value - RQ5
func setTimestampCall(newTimeStamp int64) (*string, error) {
	log.Debug("Set new timestamp via REST call")
	bodyRaw := fmt.Sprintf("{\"timestamp\" : %d}", newTimeStamp)
	log.Debugf("Body: %v", bodyRaw)
	body := []byte(bodyRaw)
	response, err := http.Post("http://localhost:10000/timestamp", "text/plain", bytes.NewBuffer(body)) // RQ3

	if err != nil {
		log.Errorf("Error when calling REST endpoint. Error: %v", err)

		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error when calling REST endpoint. Error: %v", err)

		return nil, err
	}

	responseBody := string(responseData)
	log.Debugf("Response: %v", responseBody)

	if response.StatusCode != http.StatusOK {
		log.Errorf("Unexpected HTTP response code. Status code: %v. Message: %v", response.StatusCode, responseBody)
		return &responseBody, errors.New("Unexpected HTTP response code!")
	}

	return &responseBody, nil
}

// Call REST endpoint to get current timestamp value - RQ5
func getTimestampCall() (*int64, error) {
	log.Debug("Get timestamp back via REST call")

	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodGet, "http://localhost:10000/timestamp", nil)
	request.Header.Set("content-type", "text/plain") // RQ3
	response, err := client.Do(request)

	if err != nil {
		log.Errorf("Error when calling REST endpoint. Error: %v", err)

		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error when calling REST endpoint. Error: %v", err)

		return nil, err
	}

	r := string(responseData)
	log.Debugf("Response: %v", r)

	ts := TimestampJson{}
	if err := json.Unmarshal([]byte(r), &ts); err != nil {
		log.Errorf("Unable to parse request response as a json value. Error: %v", err)

		return nil, err
	}

	return &ts.Timestamp, nil
}

func main() {
	// Initialize logger
	//log.SetLevel(log.TraceLevel)
	log.SetLevel(log.ErrorLevel) // RQ6
	log.Debug("IBM homework started")

	// Start REST server (server part)
	c := make(chan int)
	go handleRequests(c)

	// Trigger test REST calls (client part) -- RQ5

	// Let's set a dummy timestamp: current time.
	_, _ = setTimestampCall(time.Now().Unix())

	ts, _ := getTimestampCall()
	fmt.Println(*ts) // RQ6

	// Wait for the web server to be finished.
	<-c
}
