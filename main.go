// MIT License
//
// Copyright (c) [2023] [Narayan]
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// This is a TiKV API that allows you to store, retrieve, update and delete blobs.
//
// Endpoints:
//
// POST /blobs
//   - Add a new blob to the TiKV store.
//   - Request body should be a JSON object with a "blob" field.
//   - Example: {"blob": "To be or not to be, that is the question."}
//
// DELETE /blobs?blob=<query>
//   - Delete a blob from the TiKV store.
//   - Query parameter "blob" should be the exact blob to delete.
//   - Example: /blobs?blob=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20question.
//
// PUT /blobs?oldBlob=<oldBlob>&newBlob=<newBlob>
//   - Update a blob in the TiKV store.
//   - Query parameter "oldBlob" should be the exact blob to update.
//   - Query parameter "newBlob" should be the new blob to replace the old blob.
//   - Example: /blobs?oldBlob=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20question.&newBlob=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20answer.
//
// GET /?action=count
//   - Get the number of blobs in the TiKV store.
//
// GET /?action=<random>
//   - Get a random blob from the TiKV store.
//
// GET /?action=all
//   - Get all blobs from the TiKV store.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
)

const ClientPoolSize = 10
const DefaultMonitoringInterval = 30 * time.Second
const LogFile = "tikvApi.log"

var clientPool chan RawKVClientInterface
var ctx = context.Background()
var pdAddrs = []string{"pd-server:2379"}
var security = config.Security{}

// main is the entry point of the TikvApi application. It sets up logging and monitoring,
// creates a pool of TiKV clients, and handles HTTP requests for retrieving, saving, and deleting blobs.
// It uses the rawkv package to interact with TiKV.
func main() {
	setupLogging(LogFile)
	clientPool := setupClientPool(false) // not mock
	setupMonitoring(clientPool)

	mux := setupServer(clientPool)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func setupServer(clientPool chan RawKVClientInterface) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, clientPool)
	})
	return mux
}

// setupClientPool creates a pool of TiKV clients and returns a channel of clients.
// The size of the pool is determined by the clientPoolSize variable.
// Each client is created using the rawkv.NewClient function with the provided context, PD addresses, and security options.
// If an error occurs while creating a client, the function will log a fatal error and exit.
// The function returns a channel of clients that can be used to perform operations on TiKV.
func setupClientPool(useMock bool) chan RawKVClientInterface {
	clientPool := make(chan RawKVClientInterface, ClientPoolSize)
	for i := 0; i < ClientPoolSize; i++ {
		var client RawKVClientInterface
		if useMock {
			client = NewMockRawKVClientInterface(nil) // Assuming you have the mock generated
		} else {
			actualClient, err := rawkv.NewClient(ctx, pdAddrs, security)
			if err != nil {
				log.Fatalf("Failed to create TiKV client: %v", err)
			}
			client = &RawKVClientWrapper{
				client: actualClient,
			}
		}
		clientPool <- client
	}
	return clientPool
}

func getClientFromPool(clientPool chan RawKVClientInterface) RawKVClientInterface {
	if len(clientPool) > 0 && cap(clientPool) > 0 {
		return <-clientPool
	} else {
		return nil
	}
}

// setupLogging initializes a new logger and returns it.
// The logger writes to a file named "tikvApi.log" in the current directory.
// If the file does not exist, it will be created.
// If the file already exists, new logs will be appended to the end of the file.
// The logger uses the default logger flags for log entries.
func setupLogging(logname string) *log.Logger {
	logFile, err := os.OpenFile(logname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return nil
	}
	return log.New(logFile, "", log.LstdFlags)
}

// setupMonitoring sets up a goroutine that logs the number of keys in TiKV every 30 seconds.
func setupMonitoring(clientPool chan RawKVClientInterface, interval ...time.Duration) {
	sleepDuration := DefaultMonitoringInterval
	if len(interval) > 0 {
		sleepDuration = interval[0]
	}

	go func() {
		for {
			time.Sleep(sleepDuration)
			log.Printf("Number of keys in TiKV: %d", countBlobs(<-clientPool))
		}
	}()
}

// handleRequest handles incoming HTTP requests and routes them to the appropriate handler function based on the request method.
// It also manages a pool of rawkv clients to handle the requests.
func handleRequest(w http.ResponseWriter, r *http.Request, clientPool chan RawKVClientInterface) {
	client := getClientFromPool(clientPool)

	if client == nil || cap(clientPool) == 0 {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Internal server error: clientPool empty")
		return
	}

	defer func() {
		clientPool <- client
	}()

	switch r.Method {
	case http.MethodGet:
		handleGET(w, r, client)
	case http.MethodPost:
		handlePOST(w, r, client)
	case http.MethodDelete:
		handleDELETE(w, r, client)
	case http.MethodPut:
		handlePUT(w, r, client)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method")
		return
	}
}

// Further break down each HTTP method handler into its own function, e.g.:
func handleGET(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	action := r.URL.Query().Get("action")
	log.Printf("Action: %v", action)
	if action == "count" {
		handleGETCount(w, client)
	} else if action == "all" {
		handleGETAll(w, r, client)
	} else {
		handleGETRandom(w, r, client)
	}
}

func handlePOST(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	blob := r.URL.Query().Get("blob")
	if blob == "" {
		http.Error(w, "No blob provided", http.StatusBadRequest)
		log.Println("No blob provided")
		return
	}

	// Check if the blob already exists
	keys, _, err := client.Scan(r.Context(), []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		http.Error(w, "Failed to retrieve blobs", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blobs: %v", err)
		return
	}
	for _, key := range keys {
		value, err := client.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "Failed to retrieve blob", http.StatusInternalServerError)
			log.Printf("Failed to retrieve blob: %v", err)
			return
		}
		if string(value) == blob {
			http.Error(w, "Blob already exists", http.StatusConflict)
			log.Println("Blob already exists")
			return
		}
	}

	key := fmt.Sprintf("blob:%d", time.Now().UnixNano())
	err = client.Put(r.Context(), []byte(key), []byte(blob))
	if err != nil {
		http.Error(w, "Failed to save blob", http.StatusInternalServerError)
		log.Printf("Failed to save blob: %v", err)
		return
	}

	// Return the saved blob as JSON
	resp := map[string]string{"blob": blob}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func handleDELETE(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	blob := r.URL.Query().Get("blob")
	if blob == "" {
		http.Error(w, "No blob provided", http.StatusBadRequest)
		log.Println("No blob provided")
		return
	}

	keys, _, err := client.Scan(r.Context(), []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		http.Error(w, "Failed to retrieve blobs", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blobs: %v", err)
		return
	}
	var keyToDelete []byte
	for _, key := range keys {
		value, err := client.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "Failed to retrieve blob", http.StatusInternalServerError)
			log.Printf("Failed to retrieve blob: %v", err)
			return
		}
		if string(value) == blob {
			keyToDelete = key
			break
		}
	}

	if keyToDelete == nil {
		http.Error(w, "Blob not found", http.StatusNotFound)
		log.Println("Blob not found")
		return
	}

	err = client.Delete(r.Context(), keyToDelete)
	if err != nil {
		http.Error(w, "Failed to delete blob", http.StatusInternalServerError)
		log.Printf("Failed to delete blob: %v", err)
		return
	}

	// Return success message as JSON
	resp := map[string]string{"message": "Blob deleted successfully"}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func handlePUT(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	oldBlob := r.URL.Query().Get("oldBlob")
	if oldBlob == "" {
		http.Error(w, "No old blob provided", http.StatusBadRequest)
		log.Println("No old blob provided")
		return
	}
	newBlob := r.URL.Query().Get("newBlob")
	if newBlob == "" {
		http.Error(w, "No new blob provided", http.StatusBadRequest)
		log.Println("No new blob provided")
		return
	}

	keys, _, err := client.Scan(r.Context(), []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		http.Error(w, "Failed to retrieve blobs", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blobs: %v", err)
		return
	}
	var keyToUpdate []byte
	for _, key := range keys {
		value, err := client.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "Failed to retrieve blob", http.StatusInternalServerError)
			log.Printf("Failed to retrieve blob: %v", err)
			return
		}
		if string(value) == oldBlob {
			keyToUpdate = key
			break
		}
	}

	if keyToUpdate == nil {
		http.Error(w, "Blob not found", http.StatusNotFound)
		log.Println("Blob not found")
		return
	}

	err = client.Put(r.Context(), keyToUpdate, []byte(newBlob))
	if err != nil {
		http.Error(w, "Failed to update blob", http.StatusInternalServerError)
		log.Printf("Failed to update blob: %v", err)
		return
	}

	// Return the updated blob as JSON
	resp := map[string]string{"blob": newBlob}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func handleGETCount(w http.ResponseWriter, client RawKVClientInterface) {
	count := countBlobs(client)
	resp := map[string]int{"count": count}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func handleGETAll(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	keys, _, err := client.Scan(r.Context(), []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		http.Error(w, "Failed to retrieve blobs", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blobs: %v", err)
		return
	}
	if len(keys) == 0 {
		http.Error(w, "No blobs found", http.StatusNotFound)
		log.Println("No blobs found")
		return
	}

	// Retrieve all blobs' values
	var blobs []string
	for _, key := range keys {
		value, err := client.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "Failed to retrieve blob", http.StatusInternalServerError)
			log.Printf("Failed to retrieve blob: %v", err)
			return
		}
		blobs = append(blobs, string(value))
	}

	// Return all blobs as JSON array
	resp := map[string][]string{"blobs": blobs}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func handleGETRandom(w http.ResponseWriter, r *http.Request, client RawKVClientInterface) {
	keys, _, err := client.Scan(r.Context(), []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		http.Error(w, "Failed to retrieve blobs", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blobs: %v", err)
		return
	}
	if len(keys) == 0 {
		http.Error(w, "No blobs found", http.StatusNotFound)
		log.Println("No blobs found")
		return
	}

	// Use local random generator to select a random blob
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := randGen.Intn(len(keys))
	randomKey := keys[randomIndex]
	value, err := client.Get(r.Context(), randomKey)
	if err != nil {
		http.Error(w, "Failed to retrieve blob", http.StatusInternalServerError)
		log.Printf("Failed to retrieve blob: %v", err)
		return
	}
	blob := string(value)

	// Return the blob (either provided or retrieved) as JSON
	resp := map[string]string{"blob": blob}
	jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

// Implement countBlobs function to count the number of blobs in the TiKV store.
func countBlobs(client RawKVClientInterface) int {
	if client == nil {
		log.Println("Client is nil")
		return -1
	}

	keys, _, err := client.Scan(ctx, []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		log.Printf("Failed to count blobs: %v", err)
		return -1
	}
	return len(keys)
}
