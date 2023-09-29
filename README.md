# TiKV API

## Introduction

This project provides a simple API to interact with TiKV, a distributed transactional key-value database. Using this API, users can store, retrieve, update, and delete blobs of data in TiKV. The API is built with Go and utilizes the TiKV client to interact directly with a TiKV cluster.

## Requirements

- Go 1.21 or later
- Docker and Docker Compose
- TiKV client (referenced in the `go.mod` file)

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
2. Navigate to the project directory
    ```cd tikvapi
3. Use Docker Compose to build and start the services:
    ```docker-compose up --build

## Usage of API
### Add a new blob
Add a new blob to the KV Store
```curl -X POST "http://localhost:8080/?quote=HelloWorld"

### Delete a blob
Delete a specific blob from the KV Store
```curl -X DELETE "http://localhost:8080/?quote=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20question."

### Update a blob
Update a specific blob from the KV Store
```curl -X PUT "http://localhost:8080/?oldBlob=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20question.&newBlob=To%20be%20or%20not%20to%20be%2C%20that%20is%20the%20answer."

### Get the Number of Blobs
Retrieve the number of blobs in the KV store.
```curl "http://localhost:8080/blobs/count"

### Retrieve a Random Blob
Get a random blob from the KV store..
```curl "http://localhost:8080/blobs/random"

### Retrieve all Blobs
Get all blobs from the KV Store.
```curl "http://localhost:8080/blobs/all"

## Maintainers
Narayan (@codevalley)

## Contributing
We welcome contributions to this project. Please open an issue or submit a pull request on GitHub!

## License
This project is licensed under the MIT License. See the LICENSE file for details.




