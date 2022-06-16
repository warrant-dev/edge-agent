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
