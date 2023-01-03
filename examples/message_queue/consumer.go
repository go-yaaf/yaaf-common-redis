package main

import (
	_ "encoding/json"
	"fmt"
	cr "github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/go-yaaf/yaaf-common/messaging"
	_ "github.com/go-yaaf/yaaf-common/messaging"
	"math/rand"
	"sync"
	"time"
)

type RedisConsumer struct {
	uri   string
	name  string
	queue string
	error error
}

// NewRedisConsumer is a factory method
func NewRedisConsumer(uri string) *RedisConsumer {
	return &RedisConsumer{uri: uri, name: "demo", queue: "queue"}
}

// Name configure message queue (topic) name
func (p *RedisConsumer) Name(name string) *RedisConsumer {
	p.name = name
	return p
}

// Queue configure message queue (topic) name
func (p *RedisConsumer) Queue(queue string) *RedisConsumer {
	p.queue = queue
	return p
}

// Start the publisher
func (p *RedisConsumer) Start(wg *sync.WaitGroup) {
	if mq, err := cr.NewRedisMessageBus(p.uri); err != nil {
		p.error = err
		wg.Done()
	} else {
		go p.run(wg, mq)
	}
}

// GetError return error
func (p *RedisConsumer) GetError() error {
	return p.error
}

// Run starts the publisher
func (p *RedisConsumer) run(wg *sync.WaitGroup, mq messaging.IMessageBus) {

	if wg != nil {
		defer wg.Done()
	}

	rand.NewSource(time.Now().UnixNano())

	// Run consumer until no messages are in the queue
	for {
		if msg, err := mq.Pop(NewStatusMessage, time.Minute, p.queue); err != nil {
			fmt.Println("Error pop message:", err.Error())
			return
		} else {
			sm := msg.(*StatusMessage)
			fmt.Println(p.name, sm.MsgPayload.(*Status).NAME())

			// simulate message processing time
			ms := rand.Intn(500)
			time.Sleep(time.Millisecond * time.Duration(ms))
		}
	}
}
