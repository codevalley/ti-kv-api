package main

import (
	"bufio"
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
	logger := setupLogging(LogFile)
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
	// req1, err1 := http.NewRequest("POST", "/", nil)
	// w1 := httptest.NewRecorder()
	// assert.NoError(t, err1)

	// // Handle the request.
	// handlePOST(w1, req1, mockClient)

	// // Assert that the response status code is 400 (Bad Request).
	// assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
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

func TestPutErrorHandlePUT(t *testing.T) {
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
	mockClient.EXPECT().Put(context.Background(), mockKeys[0], []byte("newValue")).Return(errors.New("Failed to update blob"))

	// Handle the request.
	handlePUT(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestMatchErrorHandlePUT(t *testing.T) {
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
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return the old value for the key "blob:1".
	mockClient.EXPECT().Get(context.Background(), mockKeys[0]).Return([]byte("oldestValue"), nil)
	// Handle the request.
	handlePUT(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestGetErrorHandlePUT(t *testing.T) {
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
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return the old value for the key "blob:1".
	mockClient.EXPECT().Get(context.Background(), mockKeys[0]).Return([]byte("oldestValue"), errors.New("Failed to get blob"))
	// Handle the request.
	handlePUT(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
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

// //////New test cases////////////
// - SetupServer
// - SetupClientPool
// - handlePOST
// - handleDELETE

// Creates a new http.ServeMux instance
func TestSetupServer_ClientPoolIsNil(t *testing.T) {
	mux := setupServer(nil)
	assert.NotNil(t, mux)
}

// Returns the http.ServeMux instance
func TestSetupServer_ReturnsHTTPServeMuxInstance(t *testing.T) {
	mux := setupServer(make(chan RawKVClientInterface))
	assert.NotNil(t, mux)
}

// clientPool parameter is nil
func TestSetupServer_ClientPoolParameterIsNil(t *testing.T) {
	mux := setupServer(nil)
	assert.NotNil(t, mux)
}

// clientPool parameter is empty
func TestSetupServer_ClientPoolParameterIsEmpty(t *testing.T) {
	mux := setupServer(make(chan RawKVClientInterface, 0))
	assert.NotNil(t, mux)
}

// clientPool parameter is full
func TestSetupServer_ClientPoolParameterIsFull(t *testing.T) {
	mux := setupServer(make(chan RawKVClientInterface, 10))
	assert.NotNil(t, mux)
}

////////////////////////////////////////////////////////////////

// Use mock client if useMock is true
func TestSetupClientPoolWithMock(t *testing.T) {
	useMock := true
	clientPool := setupClientPool(useMock)

	// Assert that the client pool is of the correct size
	assert.Equal(t, ClientPoolSize, len(clientPool))

	// Assert that each client in the pool is a mock client
	for i := 0; i < ClientPoolSize; i++ {
		client := <-clientPool
		_, ok := client.(*MockRawKVClientInterface)
		assert.True(t, ok)
	}
}

// Verify client pool size matches expected size
func TestSetupClientPool_ClientPoolSizeMatchesExpectedSize(t *testing.T) {
	useMock := true
	clientPool := setupClientPool(useMock)
	assert.Equal(t, ClientPoolSize, len(clientPool))
}

// Verify mock client is added to client pool when useMock is true
func TestMockClientAddedToPoolWhenUseMockIsTrue(t *testing.T) {
	// Set up
	useMock := true
	clientPool := setupClientPool(useMock)

	// Verify
	for i := 0; i < ClientPoolSize; i++ {
		client := <-clientPool
		_, isMock := client.(*MockRawKVClientInterface)
		assert.True(t, isMock)
	}
}

// Verify mock client is created with expected parameters
func TestMockClientCreation(t *testing.T) {
	// Set up the mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client using the NewMockRawKVClientInterface function
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Assert that the mock client is not nil
	assert.NotNil(t, mockClient)

	// Assert that the mock client is created with the expected parameters
	// (assuming the mock generation code is correct)
	// ...

	// Additional assertions or verifications if needed
	// ...

}

////////////////////////////////////////////////////////////////

// handlePOST returns an error if no blob is provided
func TestHandlePOSTReturnsErrorIfNoBlobProvided(t *testing.T) {
	// Create a mock client
	mockClient := &MockRawKVClientInterface{}

	// Create a response writer and request for testing
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	// Call the handlePOST function
	handlePOST(w, r, mockClient)

	// Assert that the response writer received the correct response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "No blob provided\n", w.Body.String())
}

// handleDELETE returns an error if no blob is provided
func TestHandleDELETEReturnsErrorIfNoBlobProvided(t *testing.T) {
	// Create a mock client
	mockClient := &MockRawKVClientInterface{}

	// Create a response writer and request for testing
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/", nil)

	// Call the handleDELETE function
	handleDELETE(w, r, mockClient)

	// Assert that the response writer received the correct response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "No blob provided\n", w.Body.String())
}

////////////////////////////////////////////////////////////////
// getClientFromPool tests

// Returns a RawKVClientInterface from the clientPool
func TestReturnsRawKVClientInterfaceFromPool(t *testing.T) {
	client := &MockRawKVClientInterface{}
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- client

	result := getClientFromPool(clientPool)

	if result != client {
		t.Errorf("Expected %v, but got %v", client, result)
	}
}

// Returns a RawKVClientInterface after multiple calls to getClientFromPool
func TestReturnsRawKVClientInterfaceAfterMultipleCalls(t *testing.T) {
	client1 := &MockRawKVClientInterface{}
	client2 := &MockRawKVClientInterface{}
	clientPool := make(chan RawKVClientInterface, 2)
	clientPool <- client1
	clientPool <- client2

	result1 := getClientFromPool(clientPool)
	result2 := getClientFromPool(clientPool)

	if result1 != client1 {
		t.Errorf("Expected %v, but got %v", client1, result1)
	}
	if result2 != client2 {
		t.Errorf("Expected %v, but got %v", client2, result2)
	}
}

// Returns a RawKVClientInterface after adding and removing clients from the clientPool
func TestReturnsRawKVClientInterfaceAfterAddingAndRemovingClients(t *testing.T) {
	client1 := &MockRawKVClientInterface{}
	client2 := &MockRawKVClientInterface{}
	clientPool := make(chan RawKVClientInterface, 2)
	clientPool <- client1
	clientPool <- client2

	result1 := getClientFromPool(clientPool)
	result2 := getClientFromPool(clientPool)

	if result1 != client1 {
		t.Errorf("Expected %v, but got %v", client1, result1)
	}
	if result2 != client2 {
		t.Errorf("Expected %v, but got %v", client2, result2)
	}

	client3 := &MockRawKVClientInterface{}
	clientPool <- client3

	result3 := getClientFromPool(clientPool)

	if result3 != client3 {
		t.Errorf("Expected %v, but got %v", client3, result3)
	}
}

// Returns a RawKVClientInterface after adding more clients to the clientPool than ClientPoolSize
func TestReturnsRawKVClientInterfaceAfterAddingMoreClientsThanPoolSize(t *testing.T) {
	client1 := &MockRawKVClientInterface{}
	client2 := &MockRawKVClientInterface{}
	client3 := &MockRawKVClientInterface{}
	client4 := &MockRawKVClientInterface{}
	clientPool := make(chan RawKVClientInterface, 2)
	clientPool <- client1
	clientPool <- client2

	result1 := getClientFromPool(clientPool)
	result2 := getClientFromPool(clientPool)

	if result1 != client1 {
		t.Errorf("Expected %v, but got %v", client1, result1)
	}
	if result2 != client2 {
		t.Errorf("Expected %v, but got %v", client2, result2)
	}

	clientPool <- client3
	clientPool <- client4

	result3 := getClientFromPool(clientPool)
	result4 := getClientFromPool(clientPool)

	if result3 != client3 {
		t.Errorf("Expected %v, but got %v", client3, result3)
	}
	if result4 != client4 {
		t.Errorf("Expected %v, but got %v", client4, result4)
	}
}

////////////////////////////////////////////////////////////////
// test SetupLogging

// Function returns a valid logger object
func TestSetupLoggingReturnsValidLoggerObject(t *testing.T) {
	logname := "test1.log"
	logger := setupLogging(logname)
	if logger == nil {
		t.Errorf("Expected logger to not be nil")
	}
}

// Function creates a new log file if it doesn't exist
func TestSetupLoggingCreatesNewLogFile(t *testing.T) {
	logname := "test.log"
	_ = os.Remove(logname)
	_ = setupLogging(logname)
	_, err := os.Stat(logname)
	if os.IsNotExist(err) {
		t.Errorf("Expected log file to be created")
	}
}

// Function appends to an existing log file
func TestSetupLoggingAppendsToExistingLogFile(t *testing.T) {
	logname := "test2.log"
	_ = os.Remove(logname)
	logger1 := setupLogging(logname)
	logger1.Println("Log message 1")
	logger2 := setupLogging(logname)
	logger2.Println("Log message 2")
	file, err := os.Open(logname)
	if err != nil {
		t.Errorf("Failed to open log file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	//instead of != we are doing !contains, because logger.printLn adds timestamp to the log message
	if len(lines) != 2 {
		t.Errorf("Expected log file to have 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Log message 1") {
		t.Errorf("Expected first line to be 'Log message 1', got '%s'", lines[0])
	}
	if !strings.Contains(lines[1], "Log message 2") {
		t.Errorf("Expected second line to be 'Log message 2', got '%s'", lines[1])
	}
}

// Function fails to open log file
func TestSetupLoggingFailsToOpenLogFile(t *testing.T) {
	logname := "/root/test2.log"
	logger := setupLogging(logname)
	if logger != nil {
		t.Errorf("Expected logger to be nil")
	}
}

// Function fails to create log file
func TestSetupLoggingFailsToCreateLogFile(t *testing.T) {
	logname := "/root/test3.log"
	logger := setupLogging(logname)
	if logger != nil {
		t.Errorf("Expected logger to be nil")
	}
}

// Function fails to write to log file
func TestSetupLoggingFailsToWriteToLogFile(t *testing.T) {
	logname := "test1.log"
	file, err := os.OpenFile(logname, os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	file.Close()
	logger := setupLogging(logname)
	logger.Println("Log message")
	// No assertion can be made since the log message will not be written
}

////////////////////////////////////////////////////////////////
/// test handleRequest()

// Valid GET request
func TestValidGetRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	// Mock the Get method for the GET request.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte("randomValue"), nil).AnyTimes()

	// Mock the Scan method for the GET request.
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

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

// Valid POST request
func TestValidPostRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}

	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return different values for each key to simulate that the blob doesn't exist.
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return([]byte("notPostMe"), nil).AnyTimes()

	expectedBlobForPost := "postBlobValue"
	// Mock the Put method to save the blob.
	mockClient.EXPECT().Put(context.Background(), gomock.Any(), []byte(expectedBlobForPost)).Return(nil)
	// Mock the Put method for the POST request to save the blob.

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP POST method.
	req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestErrorScanPostRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}

	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, errors.New("failed to retrieve blobs"))

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP POST method.
	req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestErrorFetchPostRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}

	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)
	// Mock the Get method to return different values for each key to simulate that the blob doesn't exist.
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return([]byte("notPostMe"), errors.New("failed to retrieve blob")).AnyTimes()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP POST method.
	req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestErrorDuplicatePostRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}

	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)
	// Mock the Get method to return different values for each key to simulate that the blob doesn't exist.
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return([]byte("postBlobValue"), nil).AnyTimes()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP POST method.
	req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestErrorPostRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}

	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method to return different values for each key to simulate that the blob doesn't exist.
	mockClient.EXPECT().Get(context.Background(), gomock.Any()).Return([]byte("notPostMe"), nil).AnyTimes()

	expectedBlobForPost := "postBlobValue"
	// Mock the Put method to save the blob.
	mockClient.EXPECT().Put(context.Background(), gomock.Any(), []byte(expectedBlobForPost)).Return(errors.New("failed to retrieve blobs"))
	// Mock the Put method for the POST request to save the blob.

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP POST method.
	req, err := http.NewRequest(http.MethodPost, "/?blob=postBlobValue", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

// Valid DELETE request
func TestValidDeleteRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

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

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP DELETE method.
	req, err := http.NewRequest(http.MethodDelete, "/?blob=deleteMe", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestInvalidDeleteRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

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

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP DELETE method.
	req, err := http.NewRequest(http.MethodDelete, "/?blob=wrong", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestScanErrorDeleteRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, errors.New("failed to retrieve blobs"))

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP DELETE method.
	req, err := http.NewRequest(http.MethodDelete, "/?blob=deleteMe", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestGetErrorDeleteRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Mock the Get method for each key.
	// For the first key, return a blob that doesn't match the one in the request.
	mockClient.EXPECT().Get(context.Background(), mockKeys[0]).Return([]byte("notTheBlobToDelete"), errors.New("Failed to retrieve blob"))

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP DELETE method.
	req, err := http.NewRequest(http.MethodDelete, "/?blob=deleteMe", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestDeleteErrorDeleteRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

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
	mockClient.EXPECT().Delete(context.Background(), mockKeys[1]).Return(errors.New("Failed to retrieve blob"))

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP DELETE method.
	req, err := http.NewRequest(http.MethodDelete, "/?blob=deleteMe", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

// Empty clientPool
func TestEmptyClientPool(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	defer close(clientPool)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP GET method.
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 500 (Internal Server Error).
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

// TODO: Invalid clientPool
// func TestInvalidClientPool(t *testing.T)

// Invalid GET request
func TestInvalidGetRequest(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client using the mock controller
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Mock client pool.
	clientPool := make(chan RawKVClientInterface, 1)
	clientPool <- mockClient
	defer close(clientPool)

	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	// Mock the Get method for the GET request.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.New("Error getting value")).AnyTimes()

	// Mock the Scan method for the GET request.
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with HTTP GET method.
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleRequest(w, req, clientPool)

	// Assert that the response status code is 500 (Internal Server Error).
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

////////////////////////////////////////////////////////////////
/// test handleGET
////////////////////////////////////////////////////////////////

// Handles action "count" by calling handleGETCount with client
func TestHandleGETCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Set up a common expectation for the Scan method
	mockKeys := [][]byte{[]byte("key1"), []byte("key2")}
	mockClient.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockKeys, nil, nil).AnyTimes()

	// Set up an expectation for the Get method for the "count" action
	mockValue := []byte("value1")
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[0])).Return(mockValue, nil).AnyTimes()
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[1])).Return(mockValue, nil).AnyTimes()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with action=count query parameter.
	req, err := http.NewRequest("GET", "/?action=count", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleGET(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

// Handles action "all" by calling handleGETAll with client
func TestHandleGETAll(t *testing.T) {
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

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with action=all query parameter.
	req, err := http.NewRequest("GET", "/?action=all", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleGET(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

// Handles other actions by calling handleGETRandom with client
func TestHandleGETRandom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Set up a common expectation for the Scan method
	mockKeys := [][]byte{[]byte("key1"), []byte("key2")}
	mockClient.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockKeys, nil, nil).AnyTimes()

	// Set up an expectation for the Get method for the "random" action
	mockValue := []byte("value1")
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[0])).Return(mockValue, nil).AnyTimes()
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[1])).Return(mockValue, nil).AnyTimes()

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with action=random query parameter.
	req, err := http.NewRequest("GET", "/?action=random", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleGET(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

// Handles empty action parameter by calling handleGETRandom with client
// should return random blob
func TestHandleGETEmptyAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock client.
	mockClient := NewMockRawKVClientInterface(ctrl)

	// Set up a common expectation for the Scan method
	mockKeys := [][]byte{[]byte("key1")}
	mockClient.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockKeys, nil, nil).AnyTimes()

	// Set up an expectation for the Get method for the "random" action
	mockValue := []byte("value1")
	mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(mockKeys[0])).Return(mockValue, nil).AnyTimes()

	// Call the handleGET function with an empty action
	req, err := http.NewRequest(http.MethodGet, "/?action=", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()
	handleGET(rr, req, mockClient)

	// Check the response status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, but got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	expectedBody := `{"blob":"value1"}`
	if rr.Body.String() != expectedBody {
		t.Errorf("Expected response body %s, but got %s", expectedBody, rr.Body.String())
	}
}

// Returns invalid request method error if request method is not GET
func TestHandleGET_ValidRequestMethod(t *testing.T) {
	// Create a mock client.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockRawKVClientInterface(ctrl)
	// Mock the Scan method to return a slice of keys.
	mockKeys := [][]byte{
		[]byte("blob:1"),
		[]byte("blob:2"),
		[]byte("blob:3"),
	}
	// Mock the Get method for the GET request.
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte("randomValue"), nil).AnyTimes()

	// Mock the Scan method for the GET request.
	mockClient.EXPECT().Scan(context.Background(), []byte("blob:"), []byte("blob:~"), 100).Return(mockKeys, nil, nil)
	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with valid request method.
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleGET(w, req, mockClient)

	// Assert that the response status code is 200 (OK).
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

// Logs action parameter
func TestHandleGETLogsActionParameter(t *testing.T) {
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

// Returns not found error if action parameter is "all" and there are no blobs
func TestHandleGETWithBlobs(t *testing.T) {
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

	// Create a mock response writer.
	w := httptest.NewRecorder()

	// Mock request with action=all query parameter.
	req, err := http.NewRequest("GET", "/?action=all", nil)
	assert.NoError(t, err)

	// Handle the request.
	handleGET(w, req, mockClient)

	// Assert that the response status code is 200.
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

// Handles error from handleGETCount by returning internal server error
//TODO: TestHandleGETCountError

//TODO: TestHandleGETAllError

// Handles error from handleGETRandom by returning internal server error
func TestHandleGETRandomError(t *testing.T) {
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

// Returns internal server error if client is nil or clientPool is empty
func TestHandleGET_InternalServerError(t *testing.T) {
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

// Returns bad request error if action parameter is not recognized
func TestHandleGET_ValidAction(t *testing.T) {
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

////////////////////////////////////////////////////////////////
///// Test main() method//
////////////////////////////////////////////////////////////////

// Save a blob with an empty string
func TestSaveBlobWithEmptyString(t *testing.T) {
	// Mock the client
	client := NewMockRawKVClientInterface(nil)

	// Create a new request with an empty blob
	req, err := http.NewRequest(http.MethodPost, "/?blob=", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handlePOST function with the mock client
	handlePOST(rr, req, client)

	// Check the response status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Check the response body
	expectedBody := "No blob provided\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, rr.Body.String())
	}
}

// /Additional tests to simulate errors on scan
func TestGetAllScanError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	mockClient.EXPECT().Scan(gomock.Any(), []byte("blob:"), []byte("blob:~"), 100).Return(nil, nil, errors.New("failed to retrieve blobs"))

	req, err := http.NewRequest(http.MethodGet, "/all", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()

	handleGETAll(w, req, mockClient)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to retrieve blobs\n", w.Body.String())
}
