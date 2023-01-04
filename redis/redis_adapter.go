// Redis based implementation of IDataCache interface
//

package facilities

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/go-yaaf/yaaf-common/database"
	. "github.com/go-yaaf/yaaf-common/entity"
	. "github.com/go-yaaf/yaaf-common/messaging"
)

// region Data structure and methods  ----------------------------------------------------------------------------------

type subscriber struct {
	ps     *redis.PubSub
	topics []string
}

type RedisAdapter struct {
	rc   *redis.Client
	ctx  context.Context
	subs map[string]subscriber
	sync.RWMutex

	tmp   []byte
	tmpMu sync.Mutex
}

// NewRedisDataCache factory method for Redis IDataCache implementation
//
// param: URI - represents the redis connection string in the format of: redis://user:password@host:port
// return: IDataCache instance, error
func NewRedisDataCache(URI string) (dbs database.IDataCache, err error) {

	if options, err := redis.ParseURL(URI); err != nil {
		return nil, err
	} else {
		return &RedisAdapter{
			rc:   redis.NewClient(options),
			subs: make(map[string]subscriber),
			ctx:  context.Background(),
		}, nil

	}
}

// NewRedisMessageBus factory method for Redis IMessageBus implementation
//
// param: URI - represents the redis connection string in the format of: redis://user:password@host:port
// return: IDataCache instance, error
func NewRedisMessageBus(URI string) (mq IMessageBus, err error) {

	if options, err := redis.ParseURL(URI); err != nil {
		return nil, err
	} else {
		return &RedisAdapter{
			rc:   redis.NewClient(options),
			subs: make(map[string]subscriber),
			ctx:  context.Background(),
		}, nil
	}
}

// Ping Test connectivity for retries number of time with time interval (in seconds) between retries
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

// Close cache and free resources
func (r *RedisAdapter) Close() error {
	if r.rc != nil {
		return r.rc.Close()
	} else {
		return nil
	}
}

// endregion

// region PRIVATE SECTION ----------------------------------------------------------------------------------------------

// convert raw data to entity
func (r *RedisAdapter) rawToEntity(factory EntityFactory, bytes []byte) (Entity, error) {
	entity := factory()
	if err := json.Unmarshal(bytes, &entity); err != nil {
		return nil, err
	} else {
		return entity, nil
	}
}

// convert entity to raw data
func (r *RedisAdapter) entityToRaw(entity Entity) ([]byte, error) {
	return json.Marshal(entity)
}

// convert raw data to message
func (r *RedisAdapter) rawToMessage(factory MessageFactory, bytes []byte) (IMessage, error) {
	message := factory()
	if err := json.Unmarshal(bytes, &message); err != nil {
		return nil, err
	} else {
		return message, nil
	}
}

// convert message to raw data
func (r *RedisAdapter) messageToRaw(message IMessage) ([]byte, error) {
	return json.Marshal(message)
}

// endregion
