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
	wg.Add(2)

	// Create status message publisher
	NewStatusPublisher(bus).Name("publisher").Topic("status").Duration(time.Minute).Interval(time.Millisecond * 500).Start(wg)

	// Create and run logger consumer
	NewStatusLogger(bus).Name("logger").Topic("status").Start()

	// Create and run average aggregator consumer
	NewStatusAggregator(bus).Name("average").Topic("status").Duration(time.Minute).Interval(time.Second * 5).Start(wg)

	wg.Wait()
	logger.Info("Done")

}
