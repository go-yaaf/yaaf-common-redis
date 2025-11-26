// Structure definitions and factory method for redis implementation of IDataCache and IMessageBus
//

package facilities

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	. "github.com/go-yaaf/yaaf-common/entity"
	"github.com/go-yaaf/yaaf-common/logger"
	. "github.com/go-yaaf/yaaf-common/messaging"
)

// region Message Bus actions ------------------------------------------------------------------------------------------

// Publish publishes messages to a channel (topic).
func (r *RedisAdapter) Publish(messages ...IMessage) error {
	for _, message := range messages {
		if bytes, err := messageToRaw(message); err != nil {
			return err
		} else {
			if res := r.rc.Publish(r.ctx, message.Topic(), bytes); res.Err() != nil {
				return res.Err()
			}
		}
	}
	return nil
}

// Subscribe subscribes to topics and invokes the callback function for each received message.
// It supports pattern-based subscriptions (e.g., "my-topic-*").
// It returns a subscription ID or an error.
func (r *RedisAdapter) Subscribe(subscriberName string, factory MessageFactory, callback SubscriptionCallback, topics ...string) (string, error) {

	topicArray := make([]string, 0)

	// Check if topics include * - in this case it should be patterned subscribe
	isPattern := false
	for _, t := range topics {
		if strings.Contains(t, "*") {
			isPattern = true
		}
		topicArray = append(topicArray, t)
	}

	var ps *redis.PubSub

	if isPattern {
		ps = r.rc.PSubscribe(r.ctx, topics...)
	} else {
		ps = r.rc.Subscribe(r.ctx, topics...)
	}

	subscriptionId := NanoID()

	r.Lock()
	defer r.Unlock()
	r.subs[subscriptionId] = subscriber{ps: ps, topics: topicArray}
	go r.subscriber(ps, callback, factory)
	return subscriptionId, nil
}

// subscriber is a function running an infinite loop to get messages from a channel.
func (r *RedisAdapter) subscriber(ps *redis.PubSub, callback SubscriptionCallback, factory MessageFactory) {

LOOP:
	for {
		select {
		case m := <-ps.Channel():
			if m == nil {
				break LOOP
			}
			message := factory()
			if err := Unmarshal([]byte(m.Payload), &message); err != nil {
				continue
			} else {
				go callback(message)
			}
		}
	}
}

// Unsubscribe removes a subscription by its ID.
// It returns true if the subscription was found and removed, and false otherwise.
func (r *RedisAdapter) Unsubscribe(subscriptionId string) bool {
	r.Lock()
	defer r.Unlock()

	if v, ok := r.subs[subscriptionId]; !ok {
		return false
	} else {
		if err := v.ps.Unsubscribe(r.ctx, v.topics...); err != nil {
			logger.Warn("Unsubscribe error unsubscribe: %s\n", err.Error())
		}
		if err := v.ps.Close(); err != nil {
			logger.Warn("Unsubscribe error closing PubSub: %s\n", err.Error())
		}
		delete(r.subs, subscriptionId)
		return true
	}
}

// Push appends one or multiple messages to a queue (using LPush).
func (r *RedisAdapter) Push(messages ...IMessage) error {
	for _, message := range messages {
		if bytes, err := messageToRaw(message); err != nil {
			return err
		} else {
			if er := r.rc.LPush(r.ctx, message.Topic(), bytes).Err(); er != nil {
				return er
			}
		}
	}
	return nil
}

// Pop removes and gets the last message from a queue (using RPop).
// If a timeout is provided, it will block until a message is available or the timeout is reached (using BRPop).
func (r *RedisAdapter) Pop(factory MessageFactory, timeout time.Duration, queue ...string) (IMessage, error) {

	message := factory()

	if len(queue) == 0 {
		queue = append(queue, message.Topic())
	}

	if timeout == 0 {
		if cmd := r.rc.RPop(r.ctx, queue[0]); cmd.Err() != nil {
			return nil, cmd.Err()
		} else {
			if bytes, er := cmd.Bytes(); er != nil {
				return nil, er
			} else {
				return rawToMessage(factory, bytes)
			}
		}
	} else {
		if cmd := r.rc.BRPop(r.ctx, timeout, queue...); cmd.Err() != nil {
			return nil, cmd.Err()
		} else {
			if result, err := cmd.Result(); err != nil {
				return nil, err
			} else {
				return rawToMessage(factory, []byte(result[1]))
			}
		}
	}
}

// CreateProducer creates a message producer for a specific topic.
func (r *RedisAdapter) CreateProducer(topic string) (IMessageProducer, error) {
	return &producer{
		rc:    r.rc,
		topic: topic,
	}, nil
}

// CreateConsumer creates a message consumer for a specific topic or a pattern.
func (r *RedisAdapter) CreateConsumer(subscription string, mf MessageFactory, topics ...string) (IMessageConsumer, error) {

	topicArray := make([]string, 0)

	// Check if topics include * - in this case it should be patterned subscribe
	isPattern := false
	for _, t := range topics {
		if strings.Contains(t, "*") {
			isPattern = true
		}
		topicArray = append(topicArray, t)
	}

	var ps *redis.PubSub

	if isPattern {
		ps = r.rc.PSubscribe(r.ctx, topics...)
	} else {
		ps = r.rc.Subscribe(r.ctx, topics...)
	}

	return &consumer{
		ps:        ps,
		factory:   mf,
		isPattern: isPattern,
		topics:    topicArray,
	}, nil
}

// endregion

// region Producer actions ---------------------------------------------------------------------------------------------

// producer is a redis based implementation of the IMessageProducer interface.
type producer struct {
	rc    *redis.Client
	topic string
}

// Close is a no-op for the redis producer.
func (p *producer) Close() error {
	return nil
}

// Publish publishes messages to the producer's topic.
func (p *producer) Publish(messages ...IMessage) error {
	for _, message := range messages {
		if bytes, err := messageToRaw(message); err != nil {
			return err
		} else {
			topic := message.Topic()
			if topic == "" {
				topic = p.topic
			}
			if res := p.rc.Publish(context.Background(), topic, bytes); res.Err() != nil {
				return res.Err()
			}
		}
	}
	return nil
}

// endregion

// region Consumer methods  --------------------------------------------------------------------------------------------

// consumer is a redis based implementation of the IMessageConsumer interface.
type consumer struct {
	ps        *redis.PubSub
	factory   MessageFactory
	isPattern bool
	topics    []string
}

// Close unsubscribes the consumer from its topics.
func (p *consumer) Close() error {

	if p.ps == nil {
		return nil
	}

	if p.isPattern {
		return p.ps.PUnsubscribe(context.Background(), p.topics...)
	} else {
		return p.ps.Unsubscribe(context.Background(), p.topics...)
	}
}

// Read reads a message from the topic, blocking until a new message arrives or until the timeout expires.
// Use 0 for an unlimited timeout.
// The standard way to use Read is within an infinite loop:
//
//	for {
//		if msg, err := consumer.Read(time.Second * 5); err != nil {
//			// Handle error
//		} else {
//			// Process message in a dedicated go routine
//			go processThisMessage(msg)
//		}
//	}
func (p *consumer) Read(timeout time.Duration) (IMessage, error) {

	if timeout == 0 {
		timeout = time.Hour * 24
	}

LOOP:
	for {
		select {
		case m := <-p.ps.Channel():
			if m == nil {
				break LOOP
			}
			message := p.factory()
			if err := Unmarshal([]byte(m.Payload), &message); err != nil {
				return nil, err
			} else {
				return message, nil
			}
		case <-time.After(timeout):
			return nil, fmt.Errorf("read timeout")
		}
	}
	return nil, fmt.Errorf("read timeout")
}

// endregion
