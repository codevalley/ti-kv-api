package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// Put method returns nil error
func TestPutMethodReturnsNilError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	value := []byte("value")

	mockClient.EXPECT().Put(gomock.Any(), key, value, gomock.Any()).Return(nil)

	err := wrapper.Put(context.Background(), key, value)

	assert.NoError(t, err)
}

// Delete method returns nil error
func TestDeleteMethodReturnsNilError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")

	mockClient.EXPECT().Delete(gomock.Any(), key, gomock.Any()).Return(nil)

	err := wrapper.Delete(context.Background(), key)

	assert.NoError(t, err)
}

// Get method returns error when context is cancelled
func TestGetMethodReturnsErrorWhenContextIsCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wrapper.Get(ctx, key)

	assert.Error(t, err)
}

// Put method returns error when context is cancelled
func TestPutMethodReturnsErrorWhenContextIsCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	value := []byte("value")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wrapper.Put(ctx, key, value)

	assert.Error(t, err)
}

// Delete method returns error when context is cancelled
func TestDeleteMethodReturnsErrorWhenContextIsCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wrapper.Delete(ctx, key)

	assert.Error(t, err)
}

// Scan method returns expected values
func TestScanMethodReturnsExpectedValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	startKey := []byte("start")
	endKey := []byte("end")
	limit := 10

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(gomock.Any(), startKey, endKey, limit, gomock.Any()).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(context.Background(), startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

func TestSuccessfullyScanWithOptions(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	ctx := context.Background()
	startKey := []byte("start")
	endKey := []byte("end")
	limit := 100

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	// Act
	mockClient.EXPECT().Scan(ctx, startKey, endKey, limit).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(ctx, startKey, endKey, limit)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

func TestSuccessfullyScanWithNoOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	ctx := context.Background()
	startKey := []byte("start")
	endKey := []byte("end")
	limit := 100

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(ctx, startKey, endKey, limit).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(ctx, startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}
func TestSuccessfullyScanWithLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	ctx := context.Background()
	startKey := []byte("start")
	endKey := []byte("end")
	limit := 100

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(ctx, startKey, endKey, limit).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(ctx, startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

func TestScanWithEmptyStartKeyAndEndKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	ctx := context.Background()
	startKey := []byte("")
	endKey := []byte("")
	limit := 100

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(ctx, startKey, endKey, limit).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(ctx, startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

func TestScanWithLimitZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	ctx := context.Background()
	startKey := []byte("start")
	endKey := []byte("end")
	limit := 0

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(ctx, startKey, endKey, limit).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(ctx, startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

// Get method returns error when key is nil
func TestGetMethodReturnsErrorWhenKeyIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte(nil)

	mockClient.EXPECT().Get(gomock.Any(), key).Return(nil, errors.New("key is nil"))

	_, err := wrapper.Get(context.Background(), key)

	assert.Error(t, err)
}

// Put method returns error when key is nil
func TestPutMethodReturnsErrorWhenKeyIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte(nil)
	value := []byte("value")

	// Set expectation for Put method call
	mockClient.EXPECT().Put(gomock.Any(), key, value, gomock.Any()).Return(errors.New("key is nil"))

	err := wrapper.Put(context.Background(), key, value)

	assert.Error(t, err)
}

// Put method returns error when value is nil
func TestPutMethodReturnsErrorWhenValueIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	value := []byte(nil)

	mockClient.EXPECT().Put(gomock.Any(), key, value, gomock.Any()).Return(errors.New("value is nil"))

	err := wrapper.Put(context.Background(), key, value)

	assert.Error(t, err)
}

// Scan method returns expected values when limit is less than the number of keys
func TestScanMethodReturnsExpectedValuesWhenLimitIsLessThanNumberOfKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	startKey := []byte("start")
	endKey := []byte("end")
	limit := 2

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2")}

	mockClient.EXPECT().Scan(gomock.Any(), startKey, endKey, limit, gomock.Any()).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(context.Background(), startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

// Scan method returns expected values when limit is greater than the number of keys
func TestScanMethodReturnsExpectedValuesWhenLimitIsGreaterThanNumberOfKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	startKey := []byte("start")
	endKey := []byte("end")
	limit := 5

	expectedKeys := [][]byte{[]byte("key1"), []byte("key2"), []byte("key3")}
	expectedValues := [][]byte{[]byte("value1"), []byte("value2"), []byte("value3")}

	mockClient.EXPECT().Scan(gomock.Any(), startKey, endKey, limit, gomock.Any()).Return(expectedKeys, expectedValues, nil)

	keys, values, err := wrapper.Scan(context.Background(), startKey, endKey, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)
}

// CustomError struct returns expected error message
func TestCustomErrorReturnsExpectedMessage(t *testing.T) {
	err := &CustomError{
		message: "test error",
		code:    123,
	}

	expectedErrorMessage := fmt.Sprintf("Error code: %d, Message: %s", err.code, err.message)

	assert.Equal(t, expectedErrorMessage, err.Error())
}

// RawKVClientWrapper struct wraps RawKVClientInterface
func TestGetMethodReturnsExpectedValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	expectedValue := []byte("value")

	mockClient.EXPECT().Get(gomock.Any(), key, gomock.Any()).Return(expectedValue, nil)

	value, err := wrapper.Get(context.Background(), key)

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}

// NewRawKVClientWrapper function returns expected RawKVClientWrapper object
func TestNewRawKVClientWrapperReturnsExpectedObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	assert.NotNil(t, wrapper)
	assert.Equal(t, mockClient, wrapper.client)
}

// Get method returns expected error when underlying client returns error
func TestGetMethodReturnsExpectedErrorWhenUnderlyingClientReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	expectedError := &CustomError{
		message: "custom error",
		code:    500,
	}

	mockClient.EXPECT().Get(gomock.Any(), key, gomock.Any()).Return(nil, expectedError)

	_, err := wrapper.Get(context.Background(), key)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// Put method returns expected error when underlying client returns error
func TestPutMethodReturnsExpectedErrorWhenUnderlyingClientReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockRawKVClientInterface(ctrl)
	wrapper := NewRawKVClientWrapper(mockClient)

	key := []byte("key")
	value := []byte("value")
	expectedError := &CustomError{
		message: "custom error",
		code:    500,
	}

	mockClient.EXPECT().Put(gomock.Any(), key, value, gomock.Any()).Return(expectedError)

	err := wrapper.Put(context.Background(), key, value)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}
