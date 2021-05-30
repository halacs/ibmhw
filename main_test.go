package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize test environment by starting the web server.

	// Consider to move initialization block into TestGet and TestSetAndGet test
	// cases separately to always have a clean environment for each tests.
	// Now we are simple enough to keep it here.

	c := make(chan int)
	go handleRequests(c)

	// Run the test cases
	exitCode := m.Run()

	// At the moment there is no need to tear down the environment: web server will shutdown automatically.
	os.Exit(exitCode)
}

// Check if timestamp can be queried via REST calls.
func TestGet(t *testing.T) {
	ts, err := getTimestampCall()

	if err != nil {
		t.Fail()
		t.Errorf("Error in getTimestampCall: %v", err)
	}

	if *ts != initialUnixTimestamp {
		t.Fail()
		t.Errorf("Incorrect timestamp received: %v. Expected value: %v", *ts, initialUnixTimestamp)
	}
}

// Check if new valid timestamp values can be set via REST calls.
func TestSetAndGet(t *testing.T) {
	testTimestamps := []int64{-1906357250, -100, -50, 0, 25, 75, 1622366082, 1906357250}

	// Iterate over positive inputs
	for _, testTimestamp := range testTimestamps {
		_, err := setTimestampCall(testTimestamp)

		if err != nil {
			t.Errorf("Error in setTimestampCall: %v", err)
			t.Fail()
		}

		ts, err := getTimestampCall()

		if err != nil {
			t.Fail()
			t.Errorf("Error in getTimestampCall: %v", err)
		}

		if *ts != testTimestamp {
			t.Fail()
			t.Errorf("Incorrect timestamp received: %v", *ts)
		}
	}
}

// Check if invalid timestamp value cannot be set via REST calls.
func TestSetAndGetNegative(t *testing.T) {
	testTimestamps := []string{"-19063572.50", "-10,1", "-50f", "0E0", "apple", "tree", "1622366082fruit"}

	// Note current timestamp value
	referenceTimestamp, err := getTimestampCall()

	if err != nil {
		t.Fail()
		t.Errorf("Error happend when tried to get reference timestamp: %v", err)
	}

	// Iterate over negative inputs
	for _, testTimestamp := range testTimestamps {

		// Let's try to store an incorrect timestamp. We should get error in the response while
		// server data should not be changed.
		_, err := setTimestampCallStr(testTimestamp)

		if err == nil {
			t.Errorf("Expected 'invalid timestamp input' error did not happen!")
			t.Fail()
		}

		// Get back what timestamp is in the server.
		ts, err := getTimestampCall()

		if err != nil {
			t.Fail()
			t.Errorf("Error in getTimestampCall: %v", err)
		}

		// Timestamp got back from the server must not be changed because of the invalid input we sent.
		if *ts != *referenceTimestamp {
			t.Fail()
			t.Errorf("Incorrect timestamp received: %v", *ts)
		}
	}
}

// Call REST endpoint to set current timestamp value.
// String timestamp argument to allow non-integer timestamp values for negative test cases.
func setTimestampCallStr(newTimeStampRaw string) (*string, error) {
	bodyRaw := fmt.Sprintf("{\"timestamp\" : %v}", newTimeStampRaw)
	body := []byte(bodyRaw)
	response, err := http.Post("http://localhost:10000/timestamp", "text/plain", bytes.NewBuffer(body)) // RQ3

	if err != nil {

		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {

		return nil, err
	}

	responseBody := string(responseData)

	if response.StatusCode != http.StatusOK {

		return &responseBody, errors.New("unexpected HTTP response code")
	}

	return &responseBody, nil
}
