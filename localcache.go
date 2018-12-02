// Copyright 2018 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tieredcache

import (
	"context"
	"fmt"
	"time"

	"github.com/allegro/bigcache"
)

// LocalCache defines a remote-caching approach in which keys are stored remotely in a separate
// process.
type LocalCache struct {
	Cache          *bigcache.BigCache
	Encoder        CacheEncoder
	Metrics        CacheMetrics
	TracingEnabled bool
}

// LocalCacheConfig is the necessary configuration for instantiating a LocalCache struct
type LocalCacheConfig struct {
	Eviction       time.Duration
	TTL            time.Duration
	Shards         uint // Must be power of 2
	TracingEnabled bool
}

// NewCache constructs and returns a LocalCache given configuration
func (lcc LocalCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
) (LocalCache, error) {
	cache := LocalCache{Encoder: encoder, TracingEnabled: lcc.TracingEnabled}
	if lcc.Shards != 0 && lcc.Shards%2 != 0 {
		err := fmt.Errorf("shards must be power of 2 - %v is invalid", lcc.Shards)
		return cache, err
	}
	config := bigcache.DefaultConfig(lcc.Eviction)
	if lcc.TTL != 0 {
		config.LifeWindow = lcc.TTL
	}
	if lcc.Shards != 0 {
		config.Shards = int(lcc.Shards)
	}
	var err error
	cache.Cache, err = bigcache.NewBigCache(config)
	if metrics != nil {
		cache.Metrics = metrics
	}
	return cache, err
}

// GetBytes gets the requested bytes from local cache
func (lc LocalCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return lc.Cache.Get(key)
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (lc LocalCache) Get(ctx context.Context, key string, target interface{}) error {
	data, err := lc.GetBytes(ctx, key)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.Miss()
		} else {
			lc.Metrics.Hit()
		}
	}
	if err != nil {
		return err
	}
	return lc.Encoder.Decode(data, target)
}

// SetBytes sets the provided bytes in the local cache on the provided key
func (lc LocalCache) SetBytes(ctx context.Context, key string, value []byte) error {
	return lc.Cache.Set(key, value)
}

// Set encodes the provided value and sets it in the local cache
func (lc LocalCache) Set(ctx context.Context, key string, value interface{}) error {
	encodedData, err := lc.Encoder.Encode(value)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.SetCollision()
		} else {
			lc.Metrics.Set()
		}
	}
	if err != nil {
		return err
	}
	return lc.SetBytes(ctx, key, encodedData)
}

// Delete removes the value from local cache
func (lc LocalCache) Delete(ctx context.Context, key string) error {
	err := lc.Cache.Delete(key)
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.DeleteMiss()
		} else {
			lc.Metrics.DeleteHit()
		}
	}
	return err
}

// Purge wipes out all items in local cache
func (lc LocalCache) Purge(ctx context.Context) error {
	err := lc.Cache.Reset()
	if lc.Metrics != nil {
		if err != nil {
			lc.Metrics.PurgeMiss()
		} else {
			lc.Metrics.PurgeHit()
		}
	}
	return err
}
