package test

import (
	"fmt"
	"github.com/go-yaaf/yaaf-common-redis/redis"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedisCacheBinaryMarshal(t *testing.T) {
	skipCI(t)

	uri := fmt.Sprintf("redis://localhost:%s", dbPort)
	cache, err := facilities.NewRedisDataCache(uri)
	require.NoError(t, err)

	err = cache.Ping(5, 5)
	require.NoError(t, err)

	// Place item in the cache
	hero := list_of_heroes[1]
	err = cache.Set(hero.ID(), hero)
	require.NoError(t, err)

	// Get item from the cache
	expected, er := cache.Get(NewHero, hero.ID())
	require.NoError(t, er)

	require.Equal(t, hero, expected)

	fmt.Println("Done")
}
