package main

import (
	cr "github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/go-yaaf/yaaf-common/logger"
	"sync"
	"time"
)

const (
	redisUri = "redis://localhost:6379"
)

func init() {
	// Initialize logger
	logger.SetLevel("DEBUG")
	logger.Init()
}

func main() {

	// Create instance of message bus
	bus, err := cr.NewRedisMessageBus(redisUri)
	if err != nil {
		logger.Error("could not create message bus instance: %s", err.Error())
		return
	}
	// Try to connect to the Redis instance
	err = bus.Ping(3, 1)
	if err != nil {
		logger.Error("could not connect to the message bus, make sure the Redis instance is running: %s", err.Error())
		return
	}

	// Sync all publishers and consumers
	wg := &sync.WaitGroup{}
	wg.Add(4)

	// Create and run 2 publishers
	NewRedisPublisher(bus).Name("water_meter_1").Queue("water").Duration(time.Minute).Interval(time.Second).Start(wg)
	NewRedisPublisher(bus).Name("water_meter_2").Queue("water").Duration(time.Minute).Interval(time.Second * 2).Start(wg)

	// Create and run 2 consumers
	NewRedisConsumer(bus).Name("consumer_1").Queue("water").Start(wg)
	NewRedisConsumer(bus).Name("consumer_2").Queue("water").Start(wg)

	wg.Wait()
	logger.Info("Done")

}
