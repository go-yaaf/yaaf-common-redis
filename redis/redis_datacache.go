// Copyright (c) 2022. Motty Cohen
//
// Structure definitions and factory method for redis implementation of IDataCache and IMessageBus
//

package facilities

import (
	"time"

	. "github.com/go-yaaf/yaaf-common/entity"
)

// region Key actions ----------------------------------------------------------------------------------------------

// Get the value of a key
// TODO: add expiration time.Duration parameter
func (r *RedisAdapter) Get(factory EntityFactory, key string) (result Entity, err error) {
	var bytes []byte
	cmd := r.rc.Get(r.ctx, key)
	if err = cmd.Err(); err != nil {
		return nil, err
	} else {
		if bytes, err = cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return r.rawToEntity(factory, bytes)
		}
	}
}

// Set value of key with expiration
func (r *RedisAdapter) Set(key string, entity Entity) error {
	return r.SetWithExp(key, entity, 0)
}

// Del Delete keys
func (r *RedisAdapter) Del(keys ...string) error {
	return r.rc.Del(r.ctx, keys...).Err()
}

// MGet Get the value of all the given keys
// Deprecated: Use GetKeys instead
func (r *RedisAdapter) MGet(factory EntityFactory, keys ...string) ([]Entity, error) {
	return r.GetKeys(factory, keys...)
}

// GetKeys Get the value of all the given keys
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
			if entity, err := r.rawToEntity(factory, item.([]byte)); err == nil {
				entities = append(entities, entity)
			}
		}
		return entities, nil
	}
}

// SetNX Set the value of a key only if the key does not exist
func (r *RedisAdapter) SetNX(key string, entity Entity, expiration time.Duration) (bool, error) {
	if bytes, err := r.entityToRaw(entity); err != nil {
		return false, err
	} else {
		if cmd := r.rc.SetNX(r.ctx, key, bytes, expiration); cmd.Err() != nil {
			return false, err
		} else {
			if _, er := cmd.Result(); er != nil {
				return false, er
			} else {
				return true, nil
			}
		}
	}
}

// SetWithExp Set object value to a key with expiration
func (r *RedisAdapter) SetWithExp(key string, entity Entity, expiration time.Duration) error {
	if bytes, err := r.entityToRaw(entity); err != nil {
		return err
	} else {
		return r.rc.Set(r.ctx, key, bytes, expiration).Err()
	}
}

// Rename a key
func (r *RedisAdapter) Rename(key string, newKey string) error {
	return r.rc.Rename(r.ctx, key, newKey).Err()
}

// Scan keys from the provided cursor
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

// Exists Check if key exists
func (r *RedisAdapter) Exists(key string) (result bool, err error) {
	if cmd := r.rc.Exists(r.ctx, key); cmd.Err() != nil {
		return false, cmd.Err()
	} else {
		return cmd.Val() > 0, nil
	}
}

// endregion

// region Hash actions ---------------------------------------------------------------------------------------------

// HGet Get the value of a hash field
func (r *RedisAdapter) HGet(factory EntityFactory, key, field string) (Entity, error) {
	cmd := r.rc.HGet(r.ctx, key, field)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, err := cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return r.rawToEntity(factory, bytes)
		}
	}
}

// HKeys Get all the fields in a hash
func (r *RedisAdapter) HKeys(key string) ([]string, error) {
	if cmd := r.rc.HKeys(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		return cmd.Val(), nil
	}
}

// HGetAll Get all the fields and values in a hash
func (r *RedisAdapter) HGetAll(factory EntityFactory, key string) (map[string]Entity, error) {
	if cmd := r.rc.HGetAll(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		result := make(map[string]Entity)
		for k, str := range cmd.Val() {
			if entity, er := r.rawToEntity(factory, []byte(str)); er == nil {
				result[k] = entity
			}
		}
		return result, nil
	}
}

// HSet Set the value of a hash field
func (r *RedisAdapter) HSet(key, field string, entity Entity) error {
	if bytes, err := r.entityToRaw(entity); err != nil {
		return err
	} else {
		return r.rc.HSet(r.ctx, key, field, bytes).Err()
	}
}

// HDel Delete one or more hash fields
func (r *RedisAdapter) HDel(key string, fields ...string) error {
	return r.rc.HDel(r.ctx, key, fields...).Err()
}

// HSetNX Set the value of a key only if the key does not exist
func (r *RedisAdapter) HSetNX(key, field string, entity Entity) (bool, error) {
	if bytes, err := r.entityToRaw(entity); err != nil {
		return false, err
	} else {
		if err = r.rc.HSetNX(r.ctx, key, field, bytes).Err(); err != nil {
			return false, err
		} else {
			return true, nil
		}
	}
}

// HExists Check if key exists
func (r *RedisAdapter) HExists(key, field string) (bool, error) {
	if cmd := r.rc.HExists(r.ctx, key, field); cmd.Err() != nil {
		return false, cmd.Err()
	} else {
		return cmd.Val(), nil
	}
}

// endregion

// region List actions ---------------------------------------------------------------------------------------------

// RPush Append one or multiple values to a list
func (r *RedisAdapter) RPush(key string, value ...Entity) error {
	values := make([]interface{}, 0)
	for _, v := range value {
		values = append(values, v)
	}
	return r.rc.RPush(r.ctx, key, values...).Err()
}

// LPush Prepend one or multiple values to a list
func (r *RedisAdapter) LPush(key string, value ...Entity) error {
	values := make([]interface{}, 0)
	for _, v := range value {
		values = append(values, v)
	}
	return r.rc.LPush(r.ctx, key, values...).Err()
}

// RPop Remove and get the last element in a list
func (r *RedisAdapter) RPop(factory EntityFactory, key string) (Entity, error) {
	if cmd := r.rc.RPop(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, err := cmd.Bytes(); err != nil {
			return nil, err
		} else {
			return r.rawToEntity(factory, bytes)
		}
	}
}

// LPop Remove and get the first element in a list
func (r *RedisAdapter) LPop(factory EntityFactory, key string) (entity Entity, err error) {
	if cmd := r.rc.LPop(r.ctx, key); cmd.Err() != nil {
		return nil, cmd.Err()
	} else {
		if bytes, er := cmd.Bytes(); er != nil {
			return nil, er
		} else {
			return r.rawToEntity(factory, bytes)
		}
	}
}

// BRPop Remove and get the last element in a list or block until one is available
func (r *RedisAdapter) BRPop(factory EntityFactory, timeout time.Duration, keys ...string) (key string, entity Entity, err error) {
	if cmd := r.rc.BRPop(r.ctx, timeout, keys...); cmd.Err() != nil {
		return "", nil, err
	} else {
		if result, er := cmd.Result(); er != nil {
			return "", nil, err
		} else {
			key = result[0]
			entity, err = r.rawToEntity(factory, []byte(result[1]))
			return
		}
	}
}

// BLPop Remove and get the first element in a list or block until one is available
func (r *RedisAdapter) BLPop(factory EntityFactory, timeout time.Duration, keys ...string) (key string, entity Entity, err error) {
	if cmd := r.rc.BLPop(r.ctx, timeout, keys...); cmd.Err() != nil {
		return "", nil, err
	} else {
		if result, er := cmd.Result(); er != nil {
			return "", nil, err
		} else {
			key = result[0]
			entity, err = r.rawToEntity(factory, []byte(result[1]))
			return
		}
	}
}

// LRange Get a range of elements from list
func (r *RedisAdapter) LRange(factory EntityFactory, key string, start, stop int64) ([]Entity, error) {
	if list, err := r.rc.LRange(r.ctx, key, start, stop).Result(); err != nil {
		return nil, err
	} else {
		result := make([]Entity, 0)
		for _, str := range list {
			if entity, err := r.rawToEntity(factory, []byte(str)); err == nil {
				result = append(result, entity)
			}
		}
		return result, nil
	}
}

// LLen Get the length of a list
func (r *RedisAdapter) LLen(key string) (result int64) {
	return r.rc.LLen(r.ctx, key).Val()
}

// endregion
