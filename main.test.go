package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func handleRequests(w http.ResponseWriter, r *http.Request) {
	quote := r.URL.Query().Get("quote")
	if quote == "" {
		quote = "Hello, world!"
	}
	resp := map[string]string{"quote": quote}
	jsonResp, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func TestHandleRequests(t *testing.T) {
	// Create a new request with no quote parameter
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new recorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handler function
	handleRequests(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// Check the response body
	expectedQuote := "Hello, world!"
	expectedResp := map[string]string{"quote": expectedQuote}
	expectedJsonResp, _ := json.Marshal(expectedResp)
	if !bytes.Equal(rr.Body.Bytes(), expectedJsonResp) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedJsonResp))
	}

	// Create a new request with a quote parameter
	quote := "This is a test quote"
	req, err = http.NewRequest("GET", "/?quote="+quote, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new recorder to capture the response
	rr = httptest.NewRecorder()

	// Call the handler function
	handleRequests(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// Check the response body
	expectedResp = map[string]string{"quote": quote}
	expectedJsonResp, _ = json.Marshal(expectedResp)
	if !bytes.Equal(rr.Body.Bytes(), expectedJsonResp) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedJsonResp))
	}
}
