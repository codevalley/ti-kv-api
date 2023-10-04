package main

import (
	"context"
	"fmt"

	"github.com/tikv/client-go/v2/rawkv"
)

// RawKVClientInterface is an interface that wraps the rawkv.Client methods used in main.go
type RawKVClientInterface interface {
	Get(ctx context.Context, key []byte, options ...rawkv.RawOption) ([]byte, error)
	Put(ctx context.Context, key []byte, value []byte, options ...rawkv.RawOption) error
	Delete(ctx context.Context, key []byte, options ...rawkv.RawOption) error
	Scan(ctx context.Context, startKey []byte, endKey []byte, limit int, options ...rawkv.RawOption) ([][]byte, [][]byte, error)
}

// RawKVClientWrapper is a struct that wraps the rawkv.Client object and implements the RawKVClientInterface interface
type RawKVClientWrapper struct {
	client RawKVClientInterface
}

// Get is a method of the RawKVClientWrapper struct that calls the Get method on the underlying rawkv.Client object
func (r *RawKVClientWrapper) Get(ctx context.Context, key []byte, options ...rawkv.RawOption) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return r.client.Get(ctx, key, options...)
}

// Put is a method of the RawKVClientWrapper struct that calls the Put method on the underlying rawkv.Client object
func (r *RawKVClientWrapper) Put(ctx context.Context, key []byte, value []byte, options ...rawkv.RawOption) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return r.client.Put(ctx, key, value, options...)
}

// Delete is a method of the RawKVClientWrapper struct that calls the Delete method on the underlying rawkv.Client object
func (r *RawKVClientWrapper) Delete(ctx context.Context, key []byte, options ...rawkv.RawOption) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return r.client.Delete(ctx, key, options...)
}

// Scan is a method of the RawKVClientWrapper struct that calls the Scan method on the underlying rawkv.Client object
func (r *RawKVClientWrapper) Scan(ctx context.Context, startKey []byte, endKey []byte, limit int, options ...rawkv.RawOption) ([][]byte, [][]byte, error) {
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	return r.client.Scan(ctx, startKey, endKey, limit, options...)
}

// NewRawKVClientWrapper is a function that creates a new instance of the RawKVClientWrapper struct, wrapping the provided rawkv.Client object
func NewRawKVClientWrapper(client RawKVClientInterface) *RawKVClientWrapper {
	return &RawKVClientWrapper{
		client: client,
	}
}

// CustomError is a struct that represents a custom error with a message and code
type CustomError struct {
	message string
	code    int
}

// Error is a method of the CustomError struct that returns a formatted error message
func (e *CustomError) Error() string {
	return fmt.Sprintf("Error code: %d, Message: %s", e.code, e.message)
}
