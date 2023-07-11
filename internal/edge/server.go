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

import (
	"fmt"
	"log"
	"net/http"

	"github.com/warrant-dev/edge/internal/datastore"
	"github.com/warrant-dev/edge/internal/warrant"
)

const AUTHZ_API_VERSION = "v2"

const AUTHORIZED = "Authorized"
const NOT_AUTHORIZED = "Not Authorized"

const OP_ANYOF = "anyOf"
const OP_ALLOF = "allOf"

type ServerConfig struct {
	ApiKey     string
	Port       int
	StoreType  string
	Repository datastore.IRepository
}

type Server struct {
	config ServerConfig
}

type WarrantCheck struct {
	Op             string            `json:"op"`
	Warrants       []warrant.Warrant `json:"warrants" validate:"min=1,dive"`
	ConsistentRead bool              `json:"consistentRead"`
}

type WarrantCheckResponse struct {
	Code   int    `json:"code"`
	Result string `json:"result"`
}

func NewServer(config ServerConfig) *Server {
	return &Server{
		config: config,
	}
}

func (server *Server) check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !server.config.Repository.Ready() {
		SendErrorResponse(w, NewCacheNotReady())
		return
	}

	var warrantCheck WarrantCheck
	err := ParseJSONBody(r.Body, &warrantCheck)
	if err != nil {
		SendErrorResponse(w, err)
		return
	}

	var code int
	var result string
	switch warrantCheck.Op {
	case OP_ANYOF:
		code = http.StatusUnauthorized
		result = NOT_AUTHORIZED

		for _, wnt := range warrantCheck.Warrants {
			match, err := server.config.Repository.Get(wnt.String())
			if err != nil {
				SendErrorResponse(w, err)
				return
			}

			if match {
				code = http.StatusOK
				result = AUTHORIZED
				break
			}
		}
	case OP_ALLOF:
		code = http.StatusOK
		result = AUTHORIZED

		for _, wnt := range warrantCheck.Warrants {
			match, err := server.config.Repository.Get(wnt.String())
			if err != nil {
				SendErrorResponse(w, err)
				return
			}

			if !match {
				code = http.StatusUnauthorized
				result = NOT_AUTHORIZED
				break
			}
		}
	default:
		if warrantCheck.Op != "" {
			SendErrorResponse(w, NewInvalidParameterError("op", "must be one of anyOf or allOf"))
			return
		}

		if len(warrantCheck.Warrants) > 1 {
			SendErrorResponse(w, NewInvalidParameterError("op", "must include operator when including multiple warrants"))
			return
		}

		match, err := server.config.Repository.Get(warrantCheck.Warrants[0].String())
		if err != nil {
			SendErrorResponse(w, err)
			return
		}

		if match {
			code = http.StatusOK
			result = AUTHORIZED
		} else {
			code = http.StatusUnauthorized
			result = NOT_AUTHORIZED
		}
	}

	SendJSONResponse(w, WarrantCheckResponse{
		Code:   code,
		Result: result,
	})
}

func (server *Server) Run() {
	mux := http.NewServeMux()
	mux.Handle(fmt.Sprintf("/%s/authorize", AUTHZ_API_VERSION), loggingMiddleware(http.HandlerFunc(server.check)))

	log.Printf("Edge server serving authz requests on port %d", server.config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", server.config.Port), mux))
}
