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
	wg.Add(2)

	// Create status message publisher
	NewStatusPublisher(redisUri).Name("publisher").Topic("status").Duration(time.Minute).Interval(time.Millisecond * 500).Start(wg)

	// Create and run logger consumer
	NewStatusLogger(redisUri).Name("logger").Topic("status").Start()

	// Create and run average aggregator consumer
	NewStatusAggregator(redisUri).Name("average").Topic("status").Duration(time.Minute).Interval(time.Second * 5).Start(wg)

	wg.Wait()
	logger.Info("Done")

}
