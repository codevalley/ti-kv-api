package main

import (
	"context"

	"github.com/tikv/client-go/v2/rawkv"
)

// RawKVClientInterface is an interface that wraps the rawkv.Client methods used in main.go
type RawKVClientInterface interface {
	Get(ctx context.Context, key []byte, options ...rawkv.RawOption) ([]byte, error)
	Put(ctx context.Context, key []byte, value []byte, options ...rawkv.RawOption) error
	Delete(ctx context.Context, key []byte, options ...rawkv.RawOption) error
	Scan(ctx context.Context, startKey []byte, endKey []byte, limit int, options ...rawkv.RawOption) ([][]byte, [][]byte, error)
}

type RawKVClientWrapper struct {
	client *rawkv.Client
}

func (r *RawKVClientWrapper) Get(ctx context.Context, key []byte, options ...rawkv.RawOption) ([]byte, error) {
	return r.client.Get(ctx, key, options...)
}

func (r *RawKVClientWrapper) Put(ctx context.Context, key []byte, value []byte, options ...rawkv.RawOption) error {
	return r.client.Put(ctx, key, value, options...)
}

func (r *RawKVClientWrapper) Delete(ctx context.Context, key []byte, options ...rawkv.RawOption) error {
	return r.client.Delete(ctx, key, options...)
}

func (r *RawKVClientWrapper) Scan(ctx context.Context, startKey []byte, endKey []byte, limit int, options ...rawkv.RawOption) ([][]byte, [][]byte, error) {
	return r.client.Scan(ctx, startKey, endKey, limit, options...)
}
