// Copyright (c) 2022. Motty Cohen
//
// Structure definitions and factory method for redis implementation of IDataCache and IMessageBus
//

package facilities

import (
	"encoding/json"
	"github.com/go-redis/redis"
	// _ "github.com/go-redis/redis/v9"
	"github.com/go-yaaf/yaaf-common/logger"
	"github.com/google/uuid"
	"strings"
	"time"

	. "github.com/go-yaaf/yaaf-common/messaging"
)

// region Message Bus actions ------------------------------------------------------------------------------------------

// Publish messages to a channel (topic)
func (r *RedisAdapter) Publish(messages ...IMessage) error {
	for _, message := range messages {
		if bytes, err := r.messageToRaw(message); err != nil {
			return err
		} else {
			if res := r.rc.Publish(r.ctx, message.Topic(), bytes); res.Err() != nil {
				return res.Err()
			}
		}
	}
	return nil
}

// Subscribe on topics
func (r *RedisAdapter) Subscribe(callback SubscriptionCallback, factory MessageFactory, topics ...string) (subscriptionId string) {

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

	subscriptionId = uuid.New().String()

	r.Lock()
	defer r.Unlock()
	r.subs[subscriptionId] = subscriber{ps: ps, topics: topicArray}
	go r.subscriber(ps, callback, factory)
	return subscriptionId
}

// subscriber is a function running infinite loop to get messages from channel
func (r *RedisAdapter) subscriber(ps *redis.PubSub, callback SubscriptionCallback, factory MessageFactory) {

LOOP:
	for {
		select {
		case m := <-ps.Channel():
			if m == nil {
				break LOOP
			}
			message := factory()
			if err := json.Unmarshal([]byte(m.Payload), &message); err != nil {
				continue
			} else {
				message.(*BaseMessage).MsgTopic = m.Channel
				go callback(message)
			}
		}
	}
}

// Unsubscribe with the given subscriber id
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

// Push Append one or multiple messages to a queue
func (r *RedisAdapter) Push(messages ...IMessage) error {
	for _, message := range messages {
		if bytes, err := r.messageToRaw(message); err != nil {
			return err
		} else {
			if er := r.rc.LPush(r.ctx, message.Topic(), bytes).Err(); er != nil {
				return er
			}
		}
	}
	return nil
}

// Pop Remove and get the last message in a queue or block until timeout expires
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
				return r.rawToMessage(factory, bytes)
			}
		}
	} else {
		if cmd := r.rc.BRPop(r.ctx, timeout, queue...); cmd.Err() != nil {
			return nil, cmd.Err()
		} else {
			if result, err := cmd.Result(); err != nil {
				return nil, err
			} else {
				return r.rawToMessage(factory, []byte(result[1]))
			}
		}
	}
}

// endregion
