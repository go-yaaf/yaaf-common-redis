// Redis based implementation of IDataCache interface
//

package facilities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/go-yaaf/yaaf-common/database"
	. "github.com/go-yaaf/yaaf-common/entity"
	. "github.com/go-yaaf/yaaf-common/messaging"
)

// region Data structure and methods  ----------------------------------------------------------------------------------

// subscriber is a private struct to hold subscription details
type subscriber struct {
	ps     *redis.PubSub
	topics []string
}

// RedisAdapter is a redis based implementation of IDataCache and IMessageBus interfaces
type RedisAdapter struct {
	rc   *redis.Client
	ctx  context.Context
	subs map[string]subscriber
	sync.RWMutex

	tmp   []byte
	tmpMu sync.Mutex
	uri   string
}

// NewRedisDataCache is a factory method for the Redis IDataCache implementation.
// The URI parameter is a redis connection string in the format of: redis://user:password@host:port
// It returns a IDataCache instance or an error if the connection fails.
func NewRedisDataCache(URI string) (dbs database.IDataCache, error error) {

	if redisClient, err := getRedisClient(URI); err != nil {
		return nil, err
	} else {
		return &RedisAdapter{
			rc:   redisClient,
			subs: make(map[string]subscriber),
			ctx:  context.Background(),
			uri:  URI,
		}, nil
	}
}

// NewRedisMessageBus is a factory method for the Redis IMessageBus implementation.
// The URI parameter is a redis connection string in the format of: redis://user:password@host:port
// It returns a IMessageBus instance or an error if the connection fails.
func NewRedisMessageBus(URI string) (mq IMessageBus, error error) {

	if redisClient, err := getRedisClient(URI); err != nil {
		return nil, err
	} else {
		return &RedisAdapter{
			rc:   redisClient,
			subs: make(map[string]subscriber),
			ctx:  context.Background(),
		}, nil
	}
}

// Ping tests connectivity for a given number of retries with a specific time interval (in seconds) between retries.
func (r *RedisAdapter) Ping(retries uint, intervalInSeconds uint) error {

	if r.rc == nil {
		return fmt.Errorf("redis client not initialized")
	}

	ctx := context.Background()
	for i := 0; i < int(retries); i++ {
		status := r.rc.Ping(ctx)
		if status.Err() == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(intervalInSeconds))
	}
	return fmt.Errorf("no connection")
}

// Close disconnects the client from redis and frees up resources.
func (r *RedisAdapter) Close() error {
	if r.rc != nil {
		return r.rc.Close()
	} else {
		return nil
	}
}

// CloneDataCache creates a clone of the IDataCache instance.
func (r *RedisAdapter) CloneDataCache() (dbs database.IDataCache, err error) {
	return NewRedisDataCache(r.uri)
}

// CloneMessageBus creates a clone of the IMessageBus instance.
func (r *RedisAdapter) CloneMessageBus() (dbs IMessageBus, err error) {
	return NewRedisMessageBus(r.uri)
}

// endregion

// region PRIVATE SECTION ----------------------------------------------------------------------------------------------

// getRedisClient is a helper function to get a native redis client and provide a client name.
func getRedisClient(URI string) (*redis.Client, error) {
	if options, err := redis.ParseURL(URI); err != nil {
		return nil, err
	} else {
		// Create Redis client and set client name
		redisClient := redis.NewClient(options)

		if redisClient == nil {
			return nil, fmt.Errorf("can't create client")
		} else {
			clientName := fmt.Sprintf("_:%d", os.Getegid())
			if path, er := os.Executable(); er == nil {
				clientName = fmt.Sprintf("%s:%d", filepath.Base(path), os.Getegid())
			}
			_ = redisClient.Do(context.Background(), "CLIENT", "SETNAME", clientName)
			return redisClient, nil
		}
	}
}

// rawToEntity is a helper function to convert raw data to an entity.
func rawToEntity(factory EntityFactory, bytes []byte) (Entity, error) {
	entity := factory()
	if err := Unmarshal(bytes, &entity); err != nil {
		return nil, err
	} else {
		return entity, nil
	}
}

// entityToRaw is a helper function to convert an entity to raw data.
func entityToRaw(entity Entity) ([]byte, error) {
	return Marshal(entity)
}

// rawToMessage is a helper function to convert raw data to a message.
func rawToMessage(factory MessageFactory, bytes []byte) (IMessage, error) {
	message := factory()
	if err := Unmarshal(bytes, &message); err != nil {
		return nil, err
	} else {
		return message, nil
	}
}

// messageToRaw is a helper function to convert a message to raw data.
func messageToRaw(message IMessage) ([]byte, error) {
	return Marshal(message)
}

// isJsonString checks if the byte array represents a JSON string.
func isJsonString(bytes []byte) bool {
	if len(bytes) < 2 {
		return false
	}
	if string(bytes[0:1]) == "{" && string(bytes[len(bytes)-1:]) == "}" {
		return true
	}
	if string(bytes[0:1]) == "[" && string(bytes[len(bytes)-1:]) == "]" {
		return true
	}
	return false
}

// endregion
