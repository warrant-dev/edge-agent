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

package warrant

import (
	"fmt"
	"log"
)

type Subject struct {
	ObjectType string `json:"objectType" validate:"required"`
	ObjectId   string `json:"objectId" validate:"required"`
}

func (subject Subject) String() string {
	return fmt.Sprintf("%s:%s", subject.ObjectType, subject.ObjectId)
}

type Warrant struct {
	ObjectType string  `json:"objectType" validate:"required"`
	ObjectId   string  `json:"objectId" validate:"required"`
	Relation   string  `json:"relation" validate:"required"`
	Subject    Subject `json:"subject" validate:"required"`
}

func (warrant Warrant) String() string {
	return fmt.Sprintf("%s:%s#%s@%s", warrant.ObjectType, warrant.ObjectId, warrant.Relation, warrant.Subject)
}

// A set of Warrant strings
type WarrantSet map[string]uint16

func (set WarrantSet) Add(key string) {
	log.Printf("Adding warrant %s to WarrantSet", key)

	if count, ok := set[key]; ok {
		set[key] = count + 1
		return
	}

	set[key] = 1
}
