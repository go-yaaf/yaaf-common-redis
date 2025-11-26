// Structure definitions and factory method for redis implementation of IDataCache and IMessageBus
//

package facilities

import (
	"fmt"
	"time"

	. "github.com/go-yaaf/yaaf-common/database"
	. "github.com/go-yaaf/yaaf-common/entity"
)

// region Key actions ----------------------------------------------------------------------------------------------

// GetRaw gets the value of a key in a byte array format.
func (r *RedisAdapter) GetRaw(key string) ([]byte, error) {
	var bytes []byte
	cmd := r.rc.Get(r.ctx, key)
	if err := cmd.Err(); err != nil {
		return nil, err
	} else {
		if bytes, err = cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return bytes, nil
		}
	}
}

// Get gets the value of a key and decodes it into an entity.
func (r *RedisAdapter) Get(factory EntityFactory, key string) (Entity, error) {
	if bytes, err := r.GetRaw(key); err != nil {
		return nil, err
	} else {
		return rawToEntity(factory, bytes)
	}
}

// SetRaw sets the value of a key from a byte array, with an optional expiration.
func (r *RedisAdapter) SetRaw(key string, bytes []byte, expiration ...time.Duration) error {
	if len(expiration) > 0 {
		return r.rc.Set(r.ctx, key, bytes, expiration[0]).Err()
	} else {
		return r.rc.Set(r.ctx, key, bytes, 0).Err()
	}
}

// Set sets the value of a key from an entity, with an optional expiration.
func (r *RedisAdapter) Set(key string, entity Entity, expiration ...time.Duration) error {
	if bytes, err := entityToRaw(entity); err != nil {
		return err
	} else {
		return r.SetRaw(key, bytes, expiration...)
	}
}

// SetNX sets the value of a key from an entity only if the key does not already exist.
// It returns false if the key exists. An optional expiration can be provided.
func (r *RedisAdapter) SetNX(key string, entity Entity, expiration ...time.Duration) (bool, error) {
	if bytes, err := entityToRaw(entity); err != nil {
		return false, err
	} else {
		return r.SetRawNX(key, bytes, expiration...)
	}
}

// SetRawNX sets the value of a key from a byte array only if the key does not already exist.
// It returns false if the key exists. An optional expiration can be provided.
func (r *RedisAdapter) SetRawNX(key string, bytes []byte, expiration ...time.Duration) (bool, error) {
	var exp time.Duration = 0
	if len(expiration) > 0 {
		exp = expiration[0]
	}
	return r.rc.SetNX(r.ctx, key, bytes, exp).Result()
}

// Del deletes one or more keys.
func (r *RedisAdapter) Del(keys ...string) error {
	return r.rc.Del(r.ctx, keys...).Err()
}

// GetKeys gets the values of all the given keys as entities.
func (r *RedisAdapter) GetKeys(factory EntityFactory, keys ...string) ([]Entity, error) {
	cmd := r.rc.MGet(r.ctx, keys...)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	if list, err := cmd.Result(); err != nil {
		return nil, err
	} else {
		entities := make([]Entity, 0)
		for _, item := range list {
			if item != nil {
				if entity, err := rawToEntity(factory, item.([]byte)); err == nil {
					entities = append(entities, entity)
				}
			}
		}
		return entities, nil
	}
}

// GetRawKeys gets the raw values of all the given keys.
func (r *RedisAdapter) GetRawKeys(keys ...string) ([]Tuple[string, []byte], error) {
	cmd := r.rc.MGet(r.ctx, keys...)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	if list, err := cmd.Result(); err != nil {
		return nil, err
	} else {
		tuples := make([]Tuple[string, []byte], 0)
		for i, item := range list {
			if item != nil {
				tuple := Tuple[string, []byte]{Key: keys[i], Value: item.([]byte)}
				tuples = append(tuples, tuple)
			}
		}
		return tuples, nil
	}
}

// AddRaw sets the byte array value of a key only if the key does not exist.
func (r *RedisAdapter) AddRaw(key string, bytes []byte, expiration time.Duration) (bool, error) {
	if cmd := r.rc.SetNX(r.ctx, key, bytes, expiration); cmd.Err() != nil {
		return false, cmd.Err()
	} else {
		return cmd.Result()
	}
}

// Add sets the value of a key from an entity only if the key does not exist.
func (r *RedisAdapter) Add(key string, entity Entity, expiration time.Duration) (bool, error) {
	if bytes, err := entityToRaw(entity); err != nil {
		return false, err
	} else {
		return r.AddRaw(key, bytes, expiration)
	}
}

// Rename renames a key.
func (r *RedisAdapter) Rename(key string, newKey string) error {
	return r.rc.Rename(r.ctx, key, newKey).Err()
}

// Scan iterates through keys from the provided cursor that match a pattern.
func (r *RedisAdapter) Scan(from uint64, match string, count int64) (keys []string, cursor uint64, err error) {
	scanCmd := r.rc.Scan(r.ctx, from, match, count)
	if err = scanCmd.Err(); err != nil {
		return nil, 0, err
	} else {
		if list, cur, er := scanCmd.Result(); er != nil {
			return nil, 0, er
		} else {
			return list, cur, nil
		}
	}
}

// Exists checks if a key exists.
func (r *RedisAdapter) Exists(key string) (result bool, err error) {
	if cmd := r.rc.Exists(r.ctx, key); cmd.Err() != nil {
		return false, cmd.Err()
	} else {
		return cmd.Val() > 0, nil
	}
}

// endregion

// region Hash actions ---------------------------------------------------------------------------------------------

// HGet gets the value of a hash field and decodes it into an entity.
func (r *RedisAdapter) HGet(factory EntityFactory, key, field string) (Entity, error) {
	if bytes, err := r.HGetRaw(key, field); err != nil {
		return nil, err
	} else {
		return rawToEntity(factory, bytes)
	}
}

// HGetRaw gets the raw value of a hash field as a byte array.
func (r *RedisAdapter) HGetRaw(key, field string) ([]byte, error) {
	cmd := r.rc.HGet(r.ctx, key, field)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, err := cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return bytes, nil
		}
	}
}

// HKeys gets all the fields in a hash.
func (r *RedisAdapter) HKeys(key string) ([]string, error) {
	if cmd := r.rc.HKeys(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		return cmd.Val(), nil
	}
}

// HGetAll gets all the fields and values in a hash and decodes them into entities.
func (r *RedisAdapter) HGetAll(factory EntityFactory, key string) (map[string]Entity, error) {
	if cmd := r.rc.HGetAll(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		result := make(map[string]Entity)
		for k, str := range cmd.Val() {
			if entity, er := rawToEntity(factory, []byte(str)); er == nil {
				result[k] = entity
			}
		}
		return result, nil
	}
}

// HGetRawAll gets all the fields and their raw values in a hash.
func (r *RedisAdapter) HGetRawAll(key string) (map[string][]byte, error) {
	if cmd := r.rc.HGetAll(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		result := make(map[string][]byte)
		for k, str := range cmd.Val() {
			result[k] = []byte(str)
		}
		return result, nil
	}
}

// HSet sets the value of a hash field from an entity.
func (r *RedisAdapter) HSet(key, field string, entity Entity) error {
	if bytes, err := entityToRaw(entity); err != nil {
		return err
	} else {
		return r.rc.HSet(r.ctx, key, field, bytes).Err()
	}
}

// HSetRaw sets the raw value of a hash field from a byte array.
func (r *RedisAdapter) HSetRaw(key, field string, bytes []byte) error {
	return r.rc.HSet(r.ctx, key, field, bytes).Err()
}

// HSetNX sets the value of a hash field from an entity only if the field does not already exist.
// Returns false if the field already exists.
func (r *RedisAdapter) HSetNX(key string, field string, entity Entity) (bool, error) {
	if bytes, err := entityToRaw(entity); err != nil {
		return false, err
	} else {
		return r.rc.HSetNX(r.ctx, key, field, bytes).Result()
	}
}

// HSetRawNX sets the raw value of a hash field from a byte array only if the field does not already exist.
// Returns false if the field already exists.
func (r *RedisAdapter) HSetRawNX(key string, field string, bytes []byte) (bool, error) {
	return r.rc.HSetNX(r.ctx, key, field, bytes).Result()
}

// HDel deletes one or more hash fields.
func (r *RedisAdapter) HDel(key string, fields ...string) error {
	return r.rc.HDel(r.ctx, key, fields...).Err()
}

// HAdd sets the value of a hash field from an entity only if the field does not exist.
// This is an alias for HSetNX.
func (r *RedisAdapter) HAdd(key, field string, entity Entity) (bool, error) {
	if bytes, err := entityToRaw(entity); err != nil {
		return false, err
	} else {
		if err = r.rc.HSetNX(r.ctx, key, field, bytes).Err(); err != nil {
			return false, err
		} else {
			return true, nil
		}
	}
}

// HAddRaw sets the raw value of a hash field from a byte array only if the field does not exist.
// This is an alias for HSetRawNX.
func (r *RedisAdapter) HAddRaw(key, field string, bytes []byte) (bool, error) {
	if err := r.rc.HSetNX(r.ctx, key, field, bytes).Err(); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// HExists checks if a field exists in a hash.
func (r *RedisAdapter) HExists(key, field string) (bool, error) {
	if cmd := r.rc.HExists(r.ctx, key, field); cmd.Err() != nil {
		return false, cmd.Err()
	} else {
		return cmd.Val(), nil
	}
}

// endregion

// region List actions ---------------------------------------------------------------------------------------------

// RPush appends one or multiple values to a list.
func (r *RedisAdapter) RPush(key string, value ...Entity) error {
	values := make([]any, 0)
	for _, v := range value {
		if bytes, err := entityToRaw(v); err == nil {
			values = append(values, bytes)
		}
	}
	if len(values) > 0 {
		return r.rc.RPush(r.ctx, key, values...).Err()
	}
	return nil
}

// LPush prepends one or multiple values to a list.
func (r *RedisAdapter) LPush(key string, value ...Entity) error {
	values := make([]any, 0)
	for _, v := range value {
		if bytes, err := entityToRaw(v); err == nil {
			values = append(values, bytes)
		}
	}
	if len(values) > 0 {
		return r.rc.LPush(r.ctx, key, values...).Err()
	}
	return nil
}

// RPop removes and gets the last element in a list.
func (r *RedisAdapter) RPop(factory EntityFactory, key string) (Entity, error) {
	if cmd := r.rc.RPop(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, err := cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return rawToEntity(factory, bytes)
		}
	}
}

// LPop removes and gets the first element in a list.
func (r *RedisAdapter) LPop(factory EntityFactory, key string) (entity Entity, err error) {
	if cmd := r.rc.LPop(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, er := cmd.Bytes(); er != nil {
			return nil, er
		} else {
			return rawToEntity(factory, bytes)
		}
	}
}

// BRPop is a blocking version of RPop. It removes and gets the last element in a list,
// or blocks until one is available or the timeout is reached.
func (r *RedisAdapter) BRPop(factory EntityFactory, timeout time.Duration, keys ...string) (key string, entity Entity, err error) {
	if cmd := r.rc.BRPop(r.ctx, timeout, keys...); cmd.Err() != nil {
		return "", nil, err
	} else {
		if result, er := cmd.Result(); er != nil {
			return "", nil, err
		} else {
			key = result[0]
			entity, err = rawToEntity(factory, []byte(result[1]))
			return
		}
	}
}

// BLPop is a blocking version of LPop. It removes and gets the first element in a list,
// or blocks until one is available or the timeout is reached.
func (r *RedisAdapter) BLPop(factory EntityFactory, timeout time.Duration, keys ...string) (key string, entity Entity, err error) {
	if cmd := r.rc.BLPop(r.ctx, timeout, keys...); cmd.Err() != nil {
		return "", nil, err
	} else {
		if result, er := cmd.Result(); er != nil {
			return "", nil, err
		} else {
			key = result[0]
			entity, err = rawToEntity(factory, []byte(result[1]))
			return
		}
	}
}

// LRange gets a range of elements from a list.
func (r *RedisAdapter) LRange(factory EntityFactory, key string, start, stop int64) ([]Entity, error) {
	if list, err := r.rc.LRange(r.ctx, key, start, stop).Result(); err != nil {
		return nil, err
	} else {
		result := make([]Entity, 0)
		for _, str := range list {
			if entity, err := rawToEntity(factory, []byte(str)); err == nil {
				result = append(result, entity)
			}
		}
		return result, nil
	}
}

// LLen gets the length of a list.
func (r *RedisAdapter) LLen(key string) (result int64) {
	return r.rc.LLen(r.ctx, key).Val()
}

// endregion

// region Distribute Locker actions ------------------------------------------------------------------------------------

// ObtainLocker tries to obtain a new lock using a key with a given TTL.
// It returns an ILocker instance if the lock is obtained, or an error otherwise.
func (r *RedisAdapter) ObtainLocker(key string, ttl time.Duration) (ILocker, error) {
	// Create a random token
	token := ID()

	if ok, err := r.SetRawNX(key, []byte(token), ttl); err != nil {
		return nil, err
	} else {
		if ok {
			return &Locker{rc: r.rc, key: key, token: token}, nil
		} else {
			return nil, fmt.Errorf("locker key already exists")
		}
	}
}

// endregion
