// Integration tests of Redis data cache implementations
//

package test

import (
	"fmt"
	"github.com/go-yaaf/yaaf-common-redis/redis"
	"testing"
)

func TestRedisAdapterKey(t *testing.T) {
	skipCI(t)

	uri := fmt.Sprintf("redis://localhost:%s", "6379")
	sut, err := facilities.NewRedisDataCache(uri)
	if err != nil {
		panic(any(err))
	}

	if err = sut.Ping(5, 5); err != nil {
		fmt.Println("error pinging database")
		panic(any(err))
	}

	// Initialize database
	for _, hero := range list_of_heroes {
		if er := sut.Set(fmt.Sprintf("%s:%s", hero.TABLE(), hero.ID()), hero); er != nil {
			t.Errorf("error: %s", er.Error())
		}
	}

	// wait here
	fmt.Println("done")
}
