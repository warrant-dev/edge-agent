// Copyright 2024 WorkOS, Inc.
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

package edge

import (
	"net/http"

	"github.com/warrant-dev/warrant/pkg/service"
)

const (
	ErrorCacheNotReady = "cache_not_ready"
)

// CacheNotReady type
type CacheNotReady struct {
	*service.GenericError
}

func NewCacheNotReady() *CacheNotReady {
	return &CacheNotReady{
		GenericError: service.NewGenericError(
			"CacheNotReady",
			ErrorCacheNotReady,
			http.StatusServiceUnavailable,
			"Edge cache not ready",
		),
	}
}
