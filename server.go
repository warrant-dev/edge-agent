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

	check "github.com/warrant-dev/warrant/pkg/authz/check"
	"github.com/warrant-dev/warrant/pkg/service"
)

const (
	ApiVersion          = "v2"
	OpAnyOf             = "anyOf"
	OpAllOf             = "allOf"
	ResultAuthorized    = "Authorized"
	ResultNotAuthorized = "Not Authorized"
)

type ServerConfig struct {
	ApiKey     string
	Port       int
	StoreType  string
	Repository IRepository
}

type Server struct {
	config ServerConfig
}

func NewServer(config ServerConfig) (*Server, error) {
	return &Server{
		config: config,
	}, nil
}

func (server *Server) check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !server.config.Repository.Ready() {
		service.SendErrorResponse(w, NewCacheNotReady())
		return
	}

	var checkManySpec check.CheckManySpec
	err := service.ParseJSONBody(r.Body, &checkManySpec)
	if err != nil {
		service.SendErrorResponse(w, err)
	}

	var code int64
	var result string
	switch checkManySpec.Op {
	case OpAnyOf:
		code = http.StatusUnauthorized
		result = ResultNotAuthorized

		for _, wnt := range checkManySpec.Warrants {
			match, err := server.config.Repository.Get(wnt.String())
			if err != nil {
				service.SendErrorResponse(w, err)
				return
			}

			if match {
				code = http.StatusOK
				result = ResultAuthorized
				break
			}
		}
	case OpAllOf:
		code = http.StatusOK
		result = ResultAuthorized

		for _, wnt := range checkManySpec.Warrants {
			match, err := server.config.Repository.Get(wnt.String())
			if err != nil {
				service.SendErrorResponse(w, err)
				return
			}

			if !match {
				code = http.StatusUnauthorized
				result = ResultNotAuthorized
				break
			}
		}
	default:
		if checkManySpec.Op != "" {
			service.SendErrorResponse(w, service.NewInvalidParameterError("op", "must be one of anyOf or allOf"))
			return
		}

		if len(checkManySpec.Warrants) > 1 {
			service.SendErrorResponse(w, service.NewInvalidParameterError("op", "must include operator when including multiple warrants"))
			return
		}

		match, err := server.config.Repository.Get(checkManySpec.Warrants[0].String())
		if err != nil {
			service.SendErrorResponse(w, err)
			return
		}

		if match {
			code = http.StatusOK
			result = ResultAuthorized
		} else {
			code = http.StatusUnauthorized
			result = ResultNotAuthorized
		}
	}

	service.SendJSONResponse(w, check.CheckResultSpec{
		Code:   code,
		Result: result,
	})
}

func (server *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle(fmt.Sprintf("/%s/authorize", ApiVersion), loggingMiddleware(http.HandlerFunc(server.check)))
	mux.Handle(fmt.Sprintf("/%s/check", ApiVersion), loggingMiddleware(http.HandlerFunc(server.check)))

	log.Printf("Serving authz requests on port %d", server.config.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", server.config.Port), mux)
}
