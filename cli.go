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
	"time"

	"github.com/spf13/pflag"
)

// RegisterFlags registers RemoteCache pflags
func (rcc *RemoteCacheConfig) RegisterFlags(flags *pflag.FlagSet) {
	defaultURLs := []string{
		"127.0.0.1:7000",
		"127.0.0.1:7001",
		"127.0.0.1:7002",
		"127.0.0.1:7003",
		"127.0.0.1:7004",
		"127.0.0.1:7005",
	}
	flags.StringSliceVar(&rcc.URLs, "cache-urls", defaultURLs, "Remote Clustered Redis Cache URLs as a comma-separated list")
	flags.StringVar(&rcc.AuthToken, "cache-auth-token", "", "Redis Auth Token, If Any")
	flags.DurationVar(&rcc.Timeout, "cache-timeout", time.Duration(time.Second*5), "Remote Redis Cache Connection Timeout")
	flags.BoolVar(&rcc.TracingEnabled, "remote-cache-tracing-enabled", true, "Enable tracing on remote cache")
}

// RegisterFlags registers LocalCache pflags
func (lcc *LocalCacheConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&lcc.Eviction, "cache-eviction", time.Duration(time.Second*5), "How frequently to evict from cache")
	flags.DurationVar(&lcc.TTL, "cache-ttl", time.Duration(time.Minute*60), "Cache Entry TTL for local cache")
	flags.UintVar(&lcc.Shards, "cache-shards", 0, "Number of shards for local cluster. 0 means the program decides itself. Must be power of 2.")
	flags.BoolVar(&lcc.TracingEnabled, "local-cache-tracing-enabled", true, "Enable tracing on local cache")
}

// RegisterFlags registers TieredCache pflags
func (tcc *TieredCacheConfig) RegisterFlags(flags *pflag.FlagSet) {
	tcc.RemoteConfig.RegisterFlags(flags)
	tcc.LocalConfig.RegisterFlags(flags)
	flags.BoolVar(&tcc.TracingEnabled, "tiered-cache-tracing-enabled", true, "Enable tracing on tiered cache")
}
