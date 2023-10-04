# TiKV API

[![Go](https://github.com/codevalley/ti-kv-api/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/codevalley/ti-kv-api/actions/workflows/go.yml)
![coverage](https://raw.githubusercontent.com/codevalley/ti-kv-api/badges/.badges/main/coverage.svg)
## Introduction

This project provides a simple API to interact with TiKV, a distributed transactional key-value database. Using this API, users can store, retrieve, update, and delete blobs of data in TiKV. The API is built with Go and utilizes the TiKV client to interact directly with a TiKV cluster.

## Requirements

- Go 1.21 or later
- Docker and Docker Compose
- TiKV client (referenced in the `go.mod` file)

## Installation

Make sure you have the necessary frameworks and tools required to run the project.

* [Go installation](https://go.dev/doc/install)
* [Docker installation](https://docs.docker.com/desktop/) 

1. Clone the repository:

   ```bash
   git clone https://github.com/codevalley/ti-kv-api.git

2. Navigate to the project directory, and execute docker build
    ```cd tikvapi
    cd ti-kv-api
    docker build -t tikvapi .
    ```

3. Use Docker Compose to build and start the services

    ```shell
    docker-compose up
    ```

4. Your application can now be accessed at `http://localhost:8080`

5. To stop all services and remove associated containers, you can run

    ```docker-compose up --build
    docker-compose down
    ```

## Usage
### Add a new blob
Add a new blob to the KV Store

```
curl -X POST "http://localhost:8080/?blob=HelloWorld"
curl -X POST "http://localhost:8080/?blob=ByeUniverse"
curl -X POST "http://localhost:8080/?blob=GreetingsEarth"
```

### Delete a blob
Delete a specific blob from the KV Store

```
curl -X DELETE "http://localhost:8080/?blob=ByeUniverse"
```

### Update a blob
Update a specific blob from the KV Store

```
curl -X PUT "http://localhost:8080/?oldBlob=HelloWorld&newBlob=HelloMultiverse"
```

### Get the blob count

[Todo] Retrieve the number of blobs in the KV store.

```
curl "http://localhost:8080/blobs/count"
```

### Retreive a random blob

Retrieve a random entry from the KV store.

```
curl "http://localhost:8080/blobs/random"
```

### Retreive all blobs

[Todo] Retrieve all the blobs from the KV store.

```
curl "http://localhost:8080/blobs/all"
```

## Maintainers

Narayan ([@codevalley](https://github.com/codevalley))

## Contributing

We welcome contributions to this project. Please open an issue or submit a pull request on GitHub!

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/codevalley/ti-kv-api/blob/master/LICENSE.md) file for details.
