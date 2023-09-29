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
// GET /blobs/count
//   - Get the number of blobs in the TiKV store.
//   - Example: /blobs/count
//
// GET /blobs/random
//   - Get a random blob from the TiKV store.
//   - Example: /blobs/random
//
// GET /blobs/all
//   - Get all blobs from the TiKV store.
//   - Example: /blobs/all

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

var clientPool *rawkv.Client
var ctx = context.Background()
var pdAddrs = []string{"pd-server:2379"}
var security = config.Security{}

func init() {
	// Initialize TiKV client pool
	var err error
	clientPool, err = rawkv.NewClient(ctx, pdAddrs, security)
	if err != nil {
		log.Fatalf("Failed to create TiKV client: %v", err)
	}

	// Create local random generator
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	// TODO: Local random generator is not truly random
	randGen.Seed(randGen.Int63())
}

// main is the entry point of the TikvApi application. It sets up logging and monitoring,
// creates a pool of TiKV clients, and handles HTTP requests for retrieving, saving, and deleting blobs.
// It uses the rawkv package to interact with TiKV.
func main() {
	// Set up logging
	logFile, err := os.OpenFile("tikvApi.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)

	// Set up monitoring
	go func() {
		for {
			time.Sleep(30 * time.Second)
			logger.Printf("Number of keys in TiKV: %d", countBlobs())
		}
	}()

	// Create a pool of TiKV clients
	clientPoolSize := 10
	clientPool := make(chan *rawkv.Client, clientPoolSize)
	for i := 0; i < clientPoolSize; i++ {
		client, err := rawkv.NewClient(ctx, pdAddrs, security)
		if err != nil {
			log.Fatalf("Failed to create TiKV client: %v", err)
		}
		clientPool <- client
	}

	// Handle requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get a client from the pool
		client := <-clientPool
		defer func() {
			// Put the client back into the pool
			clientPool <- client
		}()

		switch r.Method {
		case http.MethodGet:
			// Retrieve a random blob if no blob is provided
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
			// TODO: Local random generator is not truly random
			randGen.Seed(randGen.Int63())
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
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				log.Printf("Failed to marshal response: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResp)
			// TODO: Code does not handle errors in tiKV operations
		case http.MethodPost:
			// Save the blob to TiKV
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
			if err != nil {
				http.Error(w, "Failed to save blob", http.StatusInternalServerError)
				log.Printf("Failed to save blob: %v", err)
				return
			}

			// Return the saved blob as JSON
			resp := map[string]string{"blob": blob}
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				log.Printf("Failed to marshal response: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResp)
			// TODO: Code does not handle errors in tiKV operations
		case http.MethodDelete:
			// Delete the blob from TiKV
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
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				log.Printf("Failed to marshal response: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResp)
			// TODO: Code does not handle errors in tiKV operations
		case http.MethodPut:
			// Update the blob in TiKV
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
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				log.Printf("Failed to marshal response: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResp)
			// TODO: Code does not handle errors in tiKV operations
		default:
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			log.Println("Invalid request method")
			return
		}
	})
	// TODO: Does not handle errors in the HTTP request handling
	logger.Fatal(http.ListenAndServe(":8080", nil))
}

// Implement countBlobs function to count the number of blobs in the TiKV store.
func countBlobs() int {
	keys, _, err := clientPool.Scan(ctx, []byte("blob:"), []byte("blob:~"), 100)
	if err != nil {
		log.Printf("Failed to count blobs: %v", err)
		return -1
	}
	return len(keys)
}
