package main

import (
	"github.com/go-yaaf/yaaf-common/logger"
	"github.com/go-yaaf/yaaf-common/messaging"
	_ "github.com/go-yaaf/yaaf-common/messaging"
	"math/rand"
	"sync"
	"time"
)

type RedisConsumer struct {
	name  string
	queue string
	error error
	mq    messaging.IMessageBus
}

// NewRedisConsumer is a factory method
func NewRedisConsumer(bus messaging.IMessageBus) *RedisConsumer {
	return &RedisConsumer{mq: bus, name: "demo", queue: "queue"}
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
	go p.run(wg)
}

// GetError return error
func (p *RedisConsumer) GetError() error {
	return p.error
}

// Run starts the publisher
func (p *RedisConsumer) run(wg *sync.WaitGroup) {

	rand.NewSource(time.Now().UnixNano())

	// Run consumer until no messages are in the queue
	for {
		if msg, err := p.mq.Pop(NewStatusMessage, time.Minute, p.queue); err != nil {
			logger.Error("Error pop message: %s", err.Error())
			if wg != nil {
				wg.Done()
			}
			return
		} else {
			sm := msg.(*StatusMessage)
			logger.Info("[%s] %s", p.name, sm.Status.NAME())

			// simulate message processing time
			ms := rand.Intn(500)
			time.Sleep(time.Millisecond * time.Duration(ms))
		}
	}
}
