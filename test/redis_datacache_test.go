// Integration tests of Redis data cache implementations
//

package test

import (
	"fmt"
	"github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/go-yaaf/yaaf-common/database"
	"github.com/go-yaaf/yaaf-common/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

const (
	dbName        = "test_db"
	dbPort        = "6379"
	containerName = "test-redis"
)

// PostgresqlTestSuite creates a Redis container with data for a suite of database tests and release it when done
type RedisCacheTestSuite struct {
	suite.Suite
	containerID string
	cache       database.IDataCache
}

func TestRedisCacheTestSuite(t *testing.T) {
	skipCI(t)
	suite.Run(t, new(RedisCacheTestSuite))
}

// SetupSuite will run once when the test suite begins
func (s *RedisCacheTestSuite) SetupSuite() {

	// Create command to run postgresql container
	err := utils.DockerUtils().CreateContainer("redis:7").
		Name(containerName).
		Port(dbPort, dbPort).
		Label("env", "test").
		Run()

	assert.Nil(s.T(), err)

	// Give it 5 seconds to warm up
	time.Sleep(5 * time.Second)

	// Create and initialize
	s.cache = s.createSUT()
}

// TearDownSuite will be run once at the end of the testing suite, after all tests have been run
func (s *RedisCacheTestSuite) TearDownSuite() {
	err := utils.DockerUtils().StopContainer(containerName)
	assert.Nil(s.T(), err)
}

// createSUT creates the system-under-test which is postgresql implementation of IDatabase
func (s *RedisCacheTestSuite) createSUT() database.IDataCache {

	uri := fmt.Sprintf("redis://localhost:%s", dbPort)
	sut, err := facilities.NewRedisDataCache(uri)
	if err != nil {
		panic(any(err))
	}

	if err := sut.Ping(5, 5); err != nil {
		fmt.Println("error pinging database")
		panic(any(err))
	}

	// Initialize sut
	for _, hero := range list_of_heroes {
		if er := sut.Set(fmt.Sprintf("%s:%s", hero.TABLE(), hero.ID()), hero); er != nil {
			s.T().Errorf("error: %s", er.Error())
		}
	}
	return sut
}

// TestDatabaseSet operation
func (s *RedisCacheTestSuite) TestDataCacheSet() {

	hero := NewHero1("100", 100, "New Hero")
	heroId := fmt.Sprintf("%s:%s", hero.TABLE(), hero.ID())

	// Set data in cache
	if err := s.cache.Set(heroId, hero); err != nil {
		s.T().Errorf("error: %s", err.Error())
	}

	// Get data from cache
	if result, err := s.cache.Get(NewHero, heroId); err != nil {
		s.T().Errorf(err.Error())
	} else {
		fmt.Println(result.ID(), result.NAME())
		require.Equal(s.T(), hero.NAME(), result.NAME(), "not expected result")
	}
}

// TestDataCacheScan operation
func (s *RedisCacheTestSuite) TestDataCacheScan() {
	if keys, cur, err := s.cache.Scan(0, "", 100); err != nil {
		s.T().Errorf(err.Error())
	} else {
		fmt.Println("cursor", cur)
		for i, key := range keys {
			fmt.Println(i, key)
		}
	}
}
