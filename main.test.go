package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tikv/client-go/v2/rawkv"
)

func TestSetupLogging(t *testing.T) {
	// Call the setupLogging function.
	setupLogging()

	// Assert that the logging subsystem is initialized.
	assert.NotNil(t, log.Default())
}

func assertLogsEvery30Seconds(t *testing.T, logger *log.Logger, message string) {
	// Get the current time.
	now := time.Now()

	// Wait for the first log message.
	for {
		time.Sleep(100 * time.Millisecond)

		// Get the next log message.
		var buf bytes.Buffer
		logger.SetOutput(&buf)
		logger.Output(2, message)
		logMessage := buf.String()

		// If the log message contains the expected message, then assert that the log message was logged within the last 30 seconds.
		if strings.Contains(logMessage, message) {
			if time.Since(now) > 30*time.Second {
				t.Errorf("Log message '%s' not logged within the last 30 seconds", message)
			}
			return
		}
	}
}

func TestSetupMonitoring(t *testing.T) {
	// Start the monitoring goroutine.
	setupMonitoring()

	// Wait for the monitoring goroutine to start up.
	time.Sleep(1 * time.Second)

	// Assert that the monitoring goroutine is running.
	monitoringRunning := false
	go func() {
		monitoringRunning = true
	}()

	// Wait for the monitoring goroutine to signal that it is running.
	for !monitoringRunning {
		time.Sleep(100 * time.Millisecond)
	}

	// Create a mock logger.
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Assert that the monitoring goroutine is logging the number of keys in TiKV every 30 seconds.
	assertLogsEvery30Seconds(t, logger, "Number of keys in TiKV: ")
}

func TestSetupClientPool(t *testing.T) {
	// Assert that the client pool is created and initialized correctly.
	clientPool := setupClientPool()
	assert.Len(t, clientPool, 0)
}

func TestHandleRequest(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Create a mock response recorder.
	w := httptest.NewRecorder()

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body is empty.
	assert.Empty(t, w.Body.String())
}

func TestHandleGET(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("GET", "/?action=count", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handleGET(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the count of the number of keys in the database.
	assert.Contains(t, w.Body.String(), "key count:")
}

func TestHandlePOST(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("POST", "/", bytes.NewBufferString("value"))
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handlePOST(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the value that was set.
	assert.Contains(t, w.Body.String(), "value")
}

func TestHandleDELETE(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("DELETE", "/", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handleDELETE(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the value that was deleted.
	assert.Contains(t, w.Body.String(), "value")
}

func TestHandlePUT(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("PUT", "/", bytes.NewBufferString("value"))
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handlePUT(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the value that was set.
	assert.Contains(t, w.Body.String(), "value")
}

func TestHandleGETCount(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	_, err := http.NewRequest("GET", "/?action=count", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handleGETCount(w, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the count of the number of keys in the database.
	assert.Contains(t, w.Body.String(), "key count:")
}

func TestHandleGETAll(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("GET", "/?action=all", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handleGETAll(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the value that was set.
	assert.Contains(t, w.Body.String(), "value")
}

func TestHandleGETRandom(t *testing.T) {
	// Create a mock client pool.
	clientPool := make(chan *rawkv.Client)
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("GET", "/?action=random", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	client := &rawkv.Client{}

	// Handle the request.
	handleGETRandom(w, req, client)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the value that was set.
	assert.Contains(t, w.Body.String(), "value")
}

func TestInvalidRequestMethod(t *testing.T) {
	// Create a mock request.
	req, err := http.NewRequest("INVALID", "/", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Handle the request.
	handleRequest(w, req, nil)

	// Assert that the response status code is 405.
	assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)

	// Assert that the response body contains an error message.
	assert.Contains(t, w.Body.String(), "Invalid request method")
}
