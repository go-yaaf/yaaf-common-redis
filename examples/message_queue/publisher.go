package main

import (
	_ "encoding/json"
	cr "github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/go-yaaf/yaaf-common/messaging"
	_ "github.com/go-yaaf/yaaf-common/messaging"
	"math/rand"
	"sync"
	"time"
)

type RedisPublisher struct {
	uri      string
	name     string
	queue    string
	duration time.Duration
	interval time.Duration
	error    error
}

// NewRedisPublisher is a factory method
func NewRedisPublisher(uri string) *RedisPublisher {
	return &RedisPublisher{uri: uri, name: "demo", queue: "queue", duration: time.Minute, interval: time.Second}
}

// Name configure message queue (topic) name
func (p *RedisPublisher) Name(name string) *RedisPublisher {
	p.name = name
	return p
}

// Queue configure message queue (topic) name
func (p *RedisPublisher) Queue(queue string) *RedisPublisher {
	p.queue = queue
	return p
}

// Duration configure for how long the publisher will run
func (p *RedisPublisher) Duration(duration time.Duration) *RedisPublisher {
	p.duration = duration
	return p
}

// Interval configure the time interval between messages
func (p *RedisPublisher) Interval(interval time.Duration) *RedisPublisher {
	p.interval = interval
	return p
}

// Start the publisher
func (p *RedisPublisher) Start(wg *sync.WaitGroup) {
	if mq, err := cr.NewRedisMessageBus(p.uri); err != nil {
		p.error = err
		wg.Done()
	} else {
		go p.run(wg, mq)
	}
}

// GetError return error
func (p *RedisPublisher) GetError() error {
	return p.error
}

// Run starts the publisher
func (p *RedisPublisher) run(wg *sync.WaitGroup, mq messaging.IMessageBus) {

	if wg != nil {
		defer wg.Done()
	}

	rand.NewSource(time.Now().UnixNano())

	// Run publisher until timeout and push status message every time interval
	after := time.After(p.duration)
	for {
		select {
		case _ = <-time.Tick(p.interval):
			cpu := rand.Intn(100)
			ram := rand.Intn(100)
			message := newStatusMessage(p.queue, NewStatus1(cpu, ram).(*Status))
			if err := mq.Push(message); err != nil {
				break
			}
		case <-after:
			return
		}
	}
}
