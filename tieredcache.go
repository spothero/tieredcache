// Copyright 2020 SpotHero
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
	"sync"

	"github.com/mna/redisc"
)

// Private shared Redis connection pool. We share one connection pool amongst all caches.
var sharedCluster = struct {
	cluster    *redisc.Cluster
	onceCreate sync.Once
	onceClose  sync.Once
}{
	nil,
	sync.Once{},
	sync.Once{},
}

// Cache defines the interface for interacting with caching utilities. All derived caches
// must implement this interface
type Cache interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	Get(ctx context.Context, key string, target interface{}) error
	SetBytes(ctx context.Context, key string, value []byte) error
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Purge(ctx context.Context) error
}

// TieredCache defines a combined local and remote-caching approach in which keys are stored
// remotely in a separate process as well as cached locally. Local cache is preferred.
type TieredCache struct {
	Remote         Cache
	Local          Cache
	Metrics        CacheMetrics
	TracingEnabled bool
}

// TieredCacheConfig is the necessary configuration for instantiating a TieredCache struct
type TieredCacheConfig struct {
	RemoteConfig   RemoteCacheConfig
	LocalConfig    LocalCacheConfig
	Encoder        CacheEncoder
	TracingEnabled bool
}

// TieredCacheCreator defines an interface to create and return a Tiered Cache
type TieredCacheCreator interface {
	NewCache(
		encoder CacheEncoder,
		metrics CacheMetrics,
		localMetrics CacheMetrics,
		remoteMetrics CacheMetrics,
	) (Cache, error)
}

// NewCache constructs and returns a TieredCache given configuration
func (tcc TieredCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
	localMetrics CacheMetrics,
	remoteMetrics CacheMetrics,
) (Cache, error) {
	remote, err := tcc.RemoteConfig.NewCache(encoder, remoteMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	local, err := tcc.LocalConfig.NewCache(encoder, localMetrics)
	if err != nil {
		return TieredCache{}, err
	}
	return TieredCache{
		Remote:         remote,
		Local:          local,
		Metrics:        metrics,
		TracingEnabled: tcc.TracingEnabled,
	}, nil
}

// Close cleans up cache and removes any open connections
func (tc TieredCache) Close() {
	tc.Remote.(RemoteCache).Close()
}

// GetBytes gets the requested bytes from from tiered cache. Local first, then remote.
func (tc TieredCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	data, err := tc.Local.GetBytes(ctx, key)
	if err != nil {
		data, err = tc.Remote.GetBytes(ctx, key)
	}
	return data, err
}

// Get retrieves the value from the tiered cache, cache, decodes it, and sets the result in target.
// Local cache first, then remote. target must be a pointer.
func (tc TieredCache) Get(ctx context.Context, key string, target interface{}) error {
	err := tc.Local.Get(ctx, key, target)
	if err != nil {
		err = tc.Remote.Get(ctx, key, target)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.Miss()
		} else {
			tc.Metrics.Hit()
		}
	}
	return err
}

// SetBytes sets the provided bytes in the local and remote caches on the provided key
func (tc TieredCache) SetBytes(ctx context.Context, key string, value []byte) error {
	err := tc.Local.SetBytes(ctx, key, value)
	if err == nil {
		err = tc.Remote.SetBytes(ctx, key, value)
	}
	return err
}

// Set encodes the provided value and sets it in the local and remote cache
func (tc TieredCache) Set(ctx context.Context, key string, value interface{}) error {
	err := tc.Local.Set(ctx, key, value)
	if err == nil {
		err = tc.Remote.Set(ctx, key, value)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.SetCollision()
		} else {
			tc.Metrics.Set()
		}
	}
	return err
}

// Delete removes the value from local cache and remote cache
func (tc TieredCache) Delete(ctx context.Context, key string) error {
	err := tc.Local.Delete(ctx, key)
	if err == nil {
		err = tc.Remote.Delete(ctx, key)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.DeleteMiss()
		} else {
			tc.Metrics.DeleteHit()
		}
	}
	return err
}

// Purge wipes out all items locally, and all items under control of this cache in Redis
func (tc TieredCache) Purge(ctx context.Context) error {
	err := tc.Local.Purge(ctx)
	if err == nil {
		err = tc.Remote.Purge(ctx)
	}
	if tc.Metrics != nil {
		if err != nil {
			tc.Metrics.PurgeMiss()
		} else {
			tc.Metrics.PurgeHit()
		}
	}
	return err
}
