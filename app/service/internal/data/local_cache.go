package data

import (
	"time"

	"github.com/patrickmn/go-cache"
)

func NewSchemaLocalCache() *cache.Cache {
	c := cache.New(2*time.Second, 10*time.Minute)
	return c
}
