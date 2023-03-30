//
// Integration tests of Redis data cache implementations
//

package test

import (
	"fmt"
	"github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/go-yaaf/yaaf-common/logger"
	"github.com/go-yaaf/yaaf-common/messaging"
	"github.com/go-yaaf/yaaf-common/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

// RedisQueueTestSuite creates a Redis container with data for a suite of message queue tests and release it when done
type RedisPubSubTestSuite struct {
	suite.Suite
	containerID string
	mq          messaging.IMessageBus
}

func TestRedisPubSubTestSuite(t *testing.T) {
	skipCI(t)
	suite.Run(t, new(RedisPubSubTestSuite))
}

// SetupSuite will run once when the test suite begins
func (s *RedisPubSubTestSuite) SetupSuite() {

	// Create command to run Redis container
	err := utils.DockerUtils().CreateContainer("redis:7").
		Name(containerName).
		Port(dbPort, dbPort).
		Label("env", "test").
		Run()

	assert.Nil(s.T(), err)

	// Give it 5 seconds to warm up
	time.Sleep(5 * time.Second)

	// Create and initialize
	s.mq = s.createSUT()
}

// TearDownSuite will be run once at the end of the testing suite, after all tests have been run
func (s *RedisPubSubTestSuite) TearDownSuite() {
	err := utils.DockerUtils().StopContainer(containerName)
	assert.Nil(s.T(), err)
}

// createSUT creates the system-under-test which is postgresql implementation of IDatabase
func (s *RedisPubSubTestSuite) createSUT() messaging.IMessageBus {

	uri := fmt.Sprintf("redis://localhost:%s", dbPort)
	sut, err := facilities.NewRedisMessageBus(uri)
	if err != nil {
		panic(any(err))
	}

	if err := sut.Ping(5, 5); err != nil {
		fmt.Println("error pinging database")
		panic(any(err))
	}

	return sut
}

func (s *RedisPubSubTestSuite) TestRedisMessageBus_PubSub() {

	// Sync all publishers and consumers
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Create and run hero subscriber
	NewHeroSubscriber(s.mq).Name("logger").Topic("hero").Start()

	// Create status message publisher
	NewHeroPublisher(s.mq).Name("publisher").Topic("hero").Duration(time.Minute).Interval(time.Millisecond * 500).Start(wg)

	wg.Wait()
	logger.Info("Done")
}
