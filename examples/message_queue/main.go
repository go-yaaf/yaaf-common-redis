package main

import (
	"github.com/go-yaaf/yaaf-common/logger"
	"github.com/go-yaaf/yaaf-common/utils"
	"sync"
	"time"
)

const (
	redisName  = "redis-example"
	redisPort  = "6379"
	redisLabel = "message-queue-example"
	redisImage = "redis:7"
	redisUri   = "redis://localhost:6379"
)

func init() {
	// Initialize logger
	logger.SetLevel("DEBUG")
	logger.Init()
}

func main() {

	// Create and run Redis container
	if err := utils.DockerUtils().CreateContainer(redisImage).
		Name(redisName).
		Port(redisPort, redisPort).
		Label("env", redisLabel).
		Run(); err != nil {
		logger.Error(err.Error())
	}

	// Sync all publishers and consumers
	wg := &sync.WaitGroup{}
	wg.Add(4)

	// Create and run 2 publishers
	NewRedisPublisher(redisUri).Name("water_meter_1").Queue("water").Duration(time.Minute).Interval(time.Second).Start(wg)
	NewRedisPublisher(redisUri).Name("water_meter_2").Queue("water").Duration(time.Minute).Interval(time.Second * 2).Start(wg)

	// Create and run 2 consumers
	NewRedisConsumer(redisUri).Name("consumer_1").Queue("water").Start(wg)
	NewRedisConsumer(redisUri).Name("consumer_2").Queue("water").Start(wg)

	wg.Wait()
	logger.Info("Done")

}
