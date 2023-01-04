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

type StatusPublisher struct {
	uri      string
	name     string
	topic    string
	duration time.Duration
	interval time.Duration
	error    error
}

// NewStatusPublisher is a factory method
func NewStatusPublisher(uri string) *StatusPublisher {
	return &StatusPublisher{uri: uri, name: "demo", topic: "topic", duration: time.Hour, interval: time.Second}
}

// Name configure message queue (topic) name
func (p *StatusPublisher) Name(name string) *StatusPublisher {
	p.name = name
	return p
}

// Topic configure message topic name
func (p *StatusPublisher) Topic(topic string) *StatusPublisher {
	p.topic = topic
	return p
}

// Duration configure for how long the publisher will run
func (p *StatusPublisher) Duration(duration time.Duration) *StatusPublisher {
	p.duration = duration
	return p
}

// Interval configure the time interval between messages
func (p *StatusPublisher) Interval(interval time.Duration) *StatusPublisher {
	p.interval = interval
	return p
}

// Start the publisher
func (p *StatusPublisher) Start(wg *sync.WaitGroup) {
	if mq, err := cr.NewRedisMessageBus(p.uri); err != nil {
		p.error = err
		wg.Done()
	} else {
		go p.run(wg, mq)
	}
}

// GetError return error
func (p *StatusPublisher) GetError() error {
	return p.error
}

// Run starts the publisher
func (p *StatusPublisher) run(wg *sync.WaitGroup, mq messaging.IMessageBus) {

	rand.NewSource(time.Now().UnixNano())

	// Run publisher until timeout and push status message every time interval
	after := time.After(p.duration)
	for {
		select {
		case _ = <-time.Tick(p.interval):
			cpu := rand.Intn(100)
			ram := rand.Intn(100)
			message := newStatusMessage(p.topic, NewStatus1(cpu, ram).(*Status))
			if err := mq.Publish(message); err != nil {
				break
			}
		case <-after:
			if wg != nil {
				wg.Done()
			}
			return
		}
	}
}
