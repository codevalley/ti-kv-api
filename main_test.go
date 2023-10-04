package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {

	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Setup the server with the mock client pool
	mux := setupServer(clientPool)
	// Create a test server using the HTTP server mux
	server := httptest.NewServer(mux)
	defer server.Close()

	//Setting the mock values correctly is most important yet painful part of this entire method.
	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(gomock.Any(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil).AnyTimes()

	// Mock the Get method for the GET request.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte("randomValue"), nil).AnyTimes()

	// Create a mock response writer.
	w := httptest.NewRecorder()
	// Mock request with HTTP GET method.
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.NoError(t, err)
	// Handle the request.
	handleRequest(w, req, clientPool)
	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
func TestHandleRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	//Setting the mock values correctly is most important yet painful part of this entire method.
	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(gomock.Any(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil).AnyTimes()

	// Mock the Get method for the GET request.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte("randomValue"), nil).AnyTimes()

	// Mock the Get method for the POST request to check if the blob exists.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.New("Blob not found")).AnyTimes()

	// Mock the Put method for the POST request to save the blob.
	expectedBlobForPost := "postBlobValue"
	mockClient.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Eq([]byte(expectedBlobForPost))).Return(nil).AnyTimes()

	// Mock the Get method for the PUT request to check if the old blob exists.
	expectedOldBlob := "oldBlobValue"
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte(expectedOldBlob), nil).AnyTimes()

	// Mock the Put method for the PUT request to update the blob.
	expectedNewBlob := "newBlobValue"
	mockClient.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Eq([]byte(expectedNewBlob))).Return(nil).AnyTimes()

	// Mock the Delete method for the DELETE request to delete the blob.
	mockClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Test for HTTP GET method
	t.Run("HTTP GET", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with HTTP GET method.
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleRequest(w, req, clientPool)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for HTTP POST method
	t.Run("HTTP POST", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with HTTP POST method.
		req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleRequest(w, req, clientPool)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for HTTP DELETE method
	t.Run("HTTP DELETE", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with HTTP DELETE method.
		req, err := http.NewRequest(http.MethodDelete, "/?blob=randomValue", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleRequest(w, req, clientPool)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for HTTP PUT method
	t.Run("HTTP PUT", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with HTTP PUT method.
		req, err := http.NewRequest(http.MethodPut, "/?oldBlob=randomValue&newBlob=newBlobValue", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleRequest(w, req, clientPool)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for invalid HTTP method
	t.Run("Invalid HTTP method", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with an invalid HTTP method.
		req, err := http.NewRequest("INVALID", "/", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleRequest(w, req, clientPool)

		// Assert that the response status code is 405 (Method Not Allowed).
		assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)
	})
}

func TestSetupLogging(t *testing.T) {
	// Call the setupLogging function.
	logger := setupLogging()
	assert.NotNil(t, logger)

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

func TestSetupClientPool(t *testing.T) {
	// Call the setupClientPool function
	clientPool := setupClientPool(true)

	// Assert that the client pool is of the correct size
	assert.Equal(t, ClientPoolSize, len(clientPool))

	// Assert that each item in the client pool is of type RawKVClientInterface
	for i := 0; i < ClientPoolSize; i++ {
		client, ok := <-clientPool
		assert.True(t, ok) // Ensure the channel is not closed
		assert.Implements(t, (*RawKVClientInterface)(nil), client)
	}
}

func TestSetupMonitoring(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Set expectations on the mock client
	mockKeys := [][]byte{[]byte("key1"), []byte("key2")}
	mockClient.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockKeys, nil, nil).Times(1)

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	// Run setupMonitoring with a short interval for testing
	setupMonitoring(clientPool, 100*time.Millisecond)

	// Sleep for a duration longer than the monitoring interval to ensure the monitoring goroutine runs
	time.Sleep(150 * time.Millisecond)

	// Check if the log contains the expected output
	expectedLog := fmt.Sprintf("Number of keys in TiKV: %d", len(mockKeys))
	if !strings.Contains(buf.String(), expectedLog) {
		t.Errorf("Expected log to contain %q, but got %q", expectedLog, buf.String())
	}
}

func TestHandleGET(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Set up a common expectation for the Scan method
	mockKeys := [][]byte{[]byte("key1"), []byte("key2")}
	mockClient.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockKeys, nil, nil).AnyTimes()

	// Set up an expectation for the Get method for the "all" action
	mockValue := []byte("value1")
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[0])).Return(mockValue, nil).AnyTimes()
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[1])).Return(mockValue, nil).AnyTimes()

	// Test for action "count"
	t.Run("action=count", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with action=count query parameter.
		req, err := http.NewRequest("GET", "/?action=count", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleGET(w, req, mockClient)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for action "all"
	t.Run("action=all", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with action=all query parameter.
		req, err := http.NewRequest("GET", "/?action=all", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleGET(w, req, mockClient)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for action "random"
	t.Run("action=random", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request with action=random query parameter.
		req, err := http.NewRequest("GET", "/?action=random", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleGET(w, req, mockClient)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	// Test for no action (defaults to "random")
	t.Run("no action", func(t *testing.T) {
		// Create a mock response writer.
		w := httptest.NewRecorder()

		// Mock request without any action query parameter.
		req, err := http.NewRequest("GET", "/", nil)
		assert.NoError(t, err)

		// Handle the request.
		handleGET(w, req, mockClient)

		// Assert that the response status code is 200.
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

func TestHandlePOST(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock request with blob query parameter.
	req, err := http.NewRequest("POST", "/?blob=postMe", nil)
	assert.NoError(t, err)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return different values for each key to simulate that the blob doesn't exist.
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return([]byte("notPostMe"), nil).AnyTimes()

	// Mock the Put method to save the blob.
	mockClient.EXPECT().Put(context.Background(), gomock.Any(), []byte("postMe")).Return(nil)

	// Handle the request.
	handlePOST(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the posted blob value.
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "postMe", resp["blob"])

	//assert scenario with no blob passed
	// Create a mock request without the blob query parameter.
	req1, err1 := http.NewRequest("POST", "/", nil)
	w1 := httptest.NewRecorder()
	assert.NoError(t, err1)

	// Handle the request.
	handlePOST(w1, req1, mockClient)

	// Assert that the response status code is 400 (Bad Request).
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleDELETE(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock request with blob query parameter.
	req, err := http.NewRequest("DELETE", "/?blob=deleteMe", nil)
	assert.NoError(t, err)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method for each key.
	// For the first key, return a blob that doesn't match the one in the request.
	mockClient.EXPECT().Get(context.Background(), mockKeys[0]).Return([]byte("notTheBlobToDelete"), nil)

	// For the second key, return the blob that matches the one in the request.
	mockClient.EXPECT().Get(context.Background(), mockKeys[1]).Return([]byte("deleteMe"), nil)

	// For the third key, return another blob that doesn't match the one in the request.
	// This expectation might not be called, so we use AnyTimes().
	mockClient.EXPECT().Get(context.Background(), mockKeys[2]).Return([]byte("anotherBlob"), nil).AnyTimes()

	// Mock the Delete method to delete the blob.
	mockClient.EXPECT().Delete(context.Background(), mockKeys[1]).Return(nil)

	// Handle the request.
	handleDELETE(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the success message.
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Blob deleted successfully", resp["message"])
}

func TestHandlePUT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock request with oldBlob and newBlob query parameters.
	req, err := http.NewRequest("PUT", "/?oldBlob=oldValue&newBlob=newValue", nil)
	assert.NoError(t, err)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return the old value for the key "blob:1".
	mockClient.EXPECT().Get(context.Background(), mockKeys[0]).Return([]byte("oldValue"), nil)

	// Mock the Put method to update the blob for the key "blob:1".
	mockClient.EXPECT().Put(context.Background(), mockKeys[0], []byte("newValue")).Return(nil)

	// Handle the request.
	handlePUT(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the updated blob value.
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "newValue", resp["blob"])
}

func TestHandleGETCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock the behavior of countBlobs function.
	// For simplicity, let's assume countBlobs uses the Scan method of the client.
	// You can adjust this based on the actual implementation of countBlobs.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Handle the request.
	handleGETCount(w, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the expected count.
	var resp map[string]int
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, len(mockKeys), resp["count"])
}

func TestHandleGETAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock request.
	req, err := http.NewRequest("GET", "/?action=all", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock the Scan method to return a slice of keys
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return a value for each key
	mockValues := [][]byte{
		[]byte("value1"),
		[]byte("value2"),
		[]byte("value3"),
	}
	for i, key := range mockKeys {
		mockClient.EXPECT().Get(context.Background(), key).Return(mockValues[i], nil)
	}

	// Handle the request.
	handleGETAll(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the mocked values.
	var resp map[string][]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"value1", "value2", "value3"}, resp["blobs"])
}

func TestHandleGETRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock request.
	req, err := http.NewRequest("GET", "/?action=random", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock the Scan method to return a slice of keys
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return a value for a random key
	mockValue := []byte("mocked value")
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return(mockValue, nil)

	// Handle the request.
	handleGETRandom(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Assert that the response body contains the mocked value.
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "mocked value", resp["blob"])
}

func TestInvalidRequestMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Create a mock request.
	req, err := http.NewRequest("INVALID", "/", nil)
	assert.NoError(t, err)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 405.
	assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)

	// Assert that the response body contains an error message.
	assert.Contains(t, w.Body.String(), "Invalid request method")
}

func TestCountBlobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock the Scan method to return a slice of keys
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Replace the global clientPool with a channel that returns the mock client
	clientPool = make(chan RawKVClientInterface, 1)
	clientPool <- mockClient

	// Call the function
	count := countBlobs(mockClient)

	// Check the result
	if count != len(mockKeys) {
		t.Errorf("Expected count to be %d, but got %d", len(mockKeys), count)
	}
}
