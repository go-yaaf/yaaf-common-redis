# GO-YAAF Redis Middleware
![Project status](https://img.shields.io/badge/version-1.2-green.svg)
[![Build](https://github.com/go-yaaf/yaaf-common-redis/actions/workflows/build.yml/badge.svg)](https://github.com/go-yaaf/yaaf-common-redis/actions/workflows/build.yml)
[![Coverage Status](https://coveralls.io/repos/go-yaaf/yaaf-common-redis/badge.svg?branch=main&service=github)](https://coveralls.io/github/go-yaaf/yaaf-common-redis?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-yaaf/yaaf-common-redis)](https://goreportcard.com/report/github.com/go-yaaf/yaaf-common-redis)
[![GoDoc](https://godoc.org/github.com/go-yaaf/yaaf-common-redis?status.svg)](https://pkg.go.dev/github.com/go-yaaf/yaaf-common-redis)
![License](https://img.shields.io/dub/l/vibe-d.svg)

## Overview

This library provides a [Redis](https://redis.io)-based implementation for two key interfaces from the `go-yaaf/yaaf-common` library:

-   `IMessageBus`: For messaging patterns like Publish/Subscribe and Message Queues.
-   `IDataCache`: For distributed data caching, supporting various data structures like Hashes, Lists, and Sets.

By using this library, you can seamlessly integrate Redis as your backend for caching and messaging within the GO-YAAF ecosystem.

## Installation

Use `go get` to install the library.

```sh
go get -v -t github.com/go-yaaf/yaaf-common-redis
```

Then import the package into your code:

```go
import facilities "github.com/go-yaaf/yaaf-common-redis/redis"
```

## Usage

### Connection String

The library uses a standard Redis connection string URI to connect to your Redis server.

**Format:** `redis://<user>:<password>@<host>:<port>/<db_number>`

-   `<user>` and `<password>` are optional.
-   `<db_number>` must be in the range of 0 - 15. If omitted, it defaults to 0.

**Examples:**
-   `redis://localhost:6379`
-   `redis://user:secret@my-redis:6379/2`

### Creating an Instance

You can create either a data cache or a message bus instance from the same connection string.

The `RedisAdapter` struct implements both `IDataCache` and `IMessageBus`, so a single instance can be used for both. `NewRedisDataCache` and `NewRedisMessageBus` return the same underlying object.

**Data Cache:**

```go
import (
    "fmt"
    "time"
    facilities "github.com/go-yaaf/yaaf-common-redis/redis"
)

func main() {
    uri := "redis://localhost:6379/0"
    dataCache, err := facilities.NewRedisDataCache(uri)
    if err != nil {
        panic(err)
    }

    // Check connectivity
    if err := dataCache.Ping(3, 1*time.Second); err != nil {
        fmt.Println("Could not connect to Redis")
    } else {
        fmt.Println("Connected to Redis!")
    }
}
```

**Message Bus:**

```go
uri := "redis://localhost:6379/0"
messageBus, err := facilities.NewRedisMessageBus(uri)
if err != nil {
    panic(err)
}
```

## Data Cache Examples

### Defining a Model

First, define a model that implements the `entity.Entity` interface from `go-yaaf/yaaf-common`.

```go
package model

import "github.com/go-yaaf/yaaf-common/entity"

// Hero represents a hero entity
type Hero struct {
    entity.BaseEntity
    Name    string `json:"name"`
    Power   string `json:"power"`
    Friends int    `json:"friends"`
}

func (h *Hero) TABLE() string { return "heroes" }

func NewHero(id, name, power string, friends int) *Hero {
	return &Hero{
		BaseEntity: entity.NewBaseEntity(id),
		Name:       name,
		Power:      power,
		Friends:    friends,
	}
}
```

### Basic Key-Value Operations

You can use `Set` and `Get` for basic CRUD operations.

```go
// Create a new hero
hero := model.NewHero("1", "superman", "flight", 100)

// Set the entity with a key and optional expiration
err := dataCache.Set("hero:1", hero, 5*time.Minute)
if err != nil {
    // handle error
}

// Get the entity
result, err := dataCache.Get(func() entity.Entity { return &model.Hero{} }, "hero:1")
if err != nil {
    // handle error
}
retrievedHero := result.(*model.Hero)
fmt.Printf("Retrieved Hero: %s\n", retrievedHero.Name)
```

### Working with Hashes

Hashes are useful for storing objects.

```go
import "github.com/go-yaaf/yaaf-common/entity"
...

// Set multiple fields in a hash
err := dataCache.HSet("hero_profile:1", "name", &entity.Any{Value: "Superman"})
err = dataCache.HSet("hero_profile:1", "power", &entity.Any{Value: "Flight"})

// Get a single field from a hash
result, err := dataCache.HGet(func() entity.Entity { return &entity.Any{} }, "hero_profile:1", "name")
name := result.(*entity.Any).Value
fmt.Println("Hero Name:", name)

// Get all fields from a hash
allFields, err := dataCache.HGetAll(func() entity.Entity { return &entity.Any{} }, "hero_profile:1")
for field, val := range allFields {
    fmt.Printf("%s: %v\n", field, val.(*entity.Any).Value)
}
```

## Message Bus Examples

### Defining a Message

Define a message that implements the `messaging.IMessage` interface.

```go
package model

import "github.com/go-yaaf/yaaf-common/messaging"

// MyMessage is a sample message
type MyMessage struct {
    messaging.BaseMessage
    Text string `json:"text"`
}

func NewMessage(topic, text string) *MyMessage {
    return &MyMessage{
        BaseMessage: messaging.NewBaseMessage(topic),
        Text:        text,
    }
}
```

### Publish/Subscribe Pattern

This pattern allows broadcasting messages to multiple subscribers on a topic.

```go
// 1. Create a message bus
messageBus, _ := facilities.NewRedisMessageBus("redis://localhost:6379")

// 2. Define a callback function for the subscriber
callback := func(msg messaging.IMessage) {
    if myMsg, ok := msg.(*model.MyMessage); ok {
        fmt.Printf("Subscriber received: %s on topic %s\n", myMsg.Text, myMsg.Topic())
    }
}

// 3. Subscribe to a topic
_, err := messageBus.Subscribe("my_subscriber", func() messaging.IMessage { return &model.MyMessage{} }, callback, "my_topic")
if err != nil {
    panic(err)
}

time.Sleep(1 * time.Second) // Wait for subscriber to be ready

// 4. Publish a message
msg := model.NewMessage("my_topic", "Hello, Pub/Sub!")
err = messageBus.Publish(msg)
if err != nil {
    panic(err)
}
```

### Message Queue Pattern

This pattern is for point-to-point messaging, where each message is processed by a single consumer.

```go
// 1. Create a message bus
messageBus, _ := facilities.NewRedisMessageBus("redis://localhost:6379")

// 2. Push a message to the queue (topic)
queueName := "my_queue"
msg := model.NewMessage(queueName, "Hello, Queue!")
err = messageBus.Push(msg)
if err != nil {
    panic(err)
}

// 3. Pop a message from the queue
// The timeout parameter determines how long to block if the queue is empty.
// A timeout of 5 seconds is used here.
retrievedMsg, err := messageBus.Pop(func() messaging.IMessage { return &model.MyMessage{} }, 5*time.Second, queueName)
if err != nil {
    panic(err)
}

if myMsg, ok := retrievedMsg.(*model.MyMessage); ok {
    fmt.Printf("Consumer received: %s from queue %s\n", myMsg.Text, myMsg.Topic())
}
```

## Further Examples

For more advanced and complete examples, please check the `examples` directory in this repository:
- [`examples/pubsub`](./examples/pubsub): A complete example of a publisher and multiple subscribers.
- [`examples/message_queue`](./examples/message_queue): A complete example of a producer and consumer for a message queue.