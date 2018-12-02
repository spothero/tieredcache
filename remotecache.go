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
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	opentracing "github.com/opentracing/opentracing-go"
)

// RemoteCache defines a remote-caching approach in which keys are stored remotely in a separate
// process.
type RemoteCache struct {
	cluster        *redisc.Cluster
	Encoder        CacheEncoder
	Metrics        CacheMetrics
	TracingEnabled bool
}

// RemoteCacheConfig is the necessary configuration for instantiating a RemoteCache struct
type RemoteCacheConfig struct {
	URLs           []string
	AuthToken      string
	Timeout        time.Duration
	TracingEnabled bool
}

// createPool creates and returns a Redis connection pool
func (rcc RemoteCacheConfig) createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     5,
		MaxActive:   10,
		IdleTimeout: time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr, opts...)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}, nil
}

// NewCache constructs and returns a RemoteCache given configuration
func (rcc RemoteCacheConfig) NewCache(
	encoder CacheEncoder,
	metrics CacheMetrics,
) (RemoteCache, error) {
	sharedCluster.onceCreate.Do(func() {
		sharedCluster.cluster = &redisc.Cluster{
			StartupNodes: rcc.URLs,
			DialOptions:  []redis.DialOption{redis.DialConnectTimeout(rcc.Timeout)},
			CreatePool:   rcc.createPool,
		}
	})
	err := sharedCluster.cluster.Refresh()
	if err == nil && rcc.AuthToken != "" {
		conn := sharedCluster.cluster.Get()
		defer conn.Close()
		_, err = conn.Do("AUTH", rcc.AuthToken)
	}
	return RemoteCache{
		cluster:        sharedCluster.cluster,
		Encoder:        encoder,
		Metrics:        metrics,
		TracingEnabled: rcc.TracingEnabled,
	}, err
}

// Close cleans up cache and removes any open connections
func (rc RemoteCache) Close() {
	sharedCluster.onceClose.Do(func() {
		sharedCluster.cluster.Close()
	})
}

// GetBytes gets the requested bytes from remote cache
func (rc RemoteCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-get-bytes")
		span.SetTag("command", "GET")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	data, err := redis.Bytes(conn.Do("GET", key))
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "miss")
		} else {
			span.SetTag("result", "hit")
		}
		span.Finish()
	}
	return data, err
}

// Get retrieves the value from cache, decodes it, and sets the result in target. target must be a
// pointer.
func (rc RemoteCache) Get(ctx context.Context, key string, target interface{}) error {
	data, err := rc.GetBytes(ctx, key)
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.Miss()
		} else {
			rc.Metrics.Hit()
		}
	}
	if err != nil {
		return err
	}
	return rc.Encoder.Decode(data, target)
}

// SetBytes sets the provided bytes in the remote cache on the provided key
func (rc RemoteCache) SetBytes(ctx context.Context, key string, value []byte) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-set-bytes")
		span.SetTag("command", "SET")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, value)
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "set")
		}
		span.Finish()
	}
	return err
}

// Set encodes the provided value and sets it in the remote cache
func (rc RemoteCache) Set(ctx context.Context, key string, value interface{}) error {
	encodedData, err := rc.Encoder.Encode(value)
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.SetCollision()
		} else {
			rc.Metrics.Set()
		}
	}
	if err != nil {
		return err
	}
	return rc.SetBytes(ctx, key, encodedData)
}

// Delete removes the value from remote cache. Because Redis doesnt support Fuzzy matches for
// delete, this function first gets all matching keys, and then proceeds to pipeline deletion of
// those keys
func (rc RemoteCache) Delete(ctx context.Context, key string) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-delete")
		span.SetTag("command", "Pipeline:KEYS:MULTI:DEL:EXEC")
		span.SetTag("key", key)
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	keysToDelete, err := redis.Strings(conn.Do("KEYS", key))

	// Execute a Redis Pipeline which bulk delets all matching keys
	err = conn.Send("MULTI")
	if rc.TracingEnabled {
		span.SetTag("num_keys", len(keysToDelete))
	}
	if rc.Metrics != nil {
		if err != nil || len(keysToDelete) <= 0 {
			rc.Metrics.DeleteMiss()
		} else {
			rc.Metrics.DeleteHit()
		}
	}
	for _, keyToDelete := range keysToDelete {
		err = conn.Send("DEL", keyToDelete)
	}
	_, err = conn.Do("EXEC")
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "delete")
		}
		span.Finish()
	}
	return err
}

// Purge wipes out all items under control of this cache in Redis
func (rc RemoteCache) Purge(ctx context.Context) error {
	var span opentracing.Span
	if rc.TracingEnabled {
		span, _ = opentracing.StartSpanFromContext(ctx, "remote-cache-delete")
		span.SetTag("command", "FLUSHALL")
	}
	conn := rc.cluster.Get()
	defer conn.Close()
	_, err := conn.Do("FLUSHALL")
	if rc.Metrics != nil {
		if err != nil {
			rc.Metrics.PurgeMiss()
		} else {
			rc.Metrics.PurgeHit()
		}
	}
	if rc.TracingEnabled {
		if err != nil {
			span.SetTag("result", "fail")
		} else {
			span.SetTag("result", "purge")
		}
		span.Finish()
	}
	return err
}
