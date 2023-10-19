// Copyright 2023 Forerunner Labs, Inc.
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

import "fmt"

type WarrantSet map[string]uint16

func (set WarrantSet) Add(key string) {
	if count, ok := set[key]; ok {
		set[key] = count + 1
		return
	}

	set[key] = 1
}

func (set WarrantSet) Has(key string) bool {
	_, exists := set[key]
	return exists
}

func (set WarrantSet) Get(key string) uint16 {
	return set[key]
}

func (set WarrantSet) String() string {
	str := ""
	for key, value := range set {
		str = fmt.Sprintf("%s\n%s => %d", str, key, value)
	}

	return str
}
