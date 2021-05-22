// Copyright 2021 The Cloud Native Events Authors
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

package restapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"sync"

	"github.com/redhat-cne/sdk-go/pkg/util/wait"

	"github.com/gorilla/mux"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/types"

	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var singleton sync.Once

// WebhookInstance ... is singleton instance
var WebhookInstance *Webhook

// Webhook defines rest routes server object
type Webhook struct {
	port       int
	apiPath    string
	closeCh    <-chan struct{}
	HTTPClient *http.Client
	httpServer *http.Server
	status     serverStatus
	targetPub  *pubsub.PubSub
}

// HwEvent ... temporary format. TODO: define schema for redfish hw event data
type HwEvent struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// InitWebhook is used to supply configurations for rest routes server
func InitWebhook(port int, apiPath string, targetPub *pubsub.PubSub) *Webhook {
	singleton.Do(func() {
		WebhookInstance = &Webhook{
			port:    port,
			apiPath: apiPath,
			status:  notReady,
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 20,
				},
				Timeout: 10 * time.Second,
			},
			targetPub: targetPub,
		}
	})
	// singleton
	return WebhookInstance
}

// EndPointHealthChk checks for rest service health
func (s *Webhook) EndPointHealthChk() (err error) {
	log.Info("checking for rest service health\n")
	for i := 0; i <= 5; i++ {
		if !s.Ready() {
			time.Sleep(healthCheckPause)
			log.Printf("server status %t", s.Ready())
			continue
		}

		log.Debugf("health check %s%s ", s.GetHostPath(), "health")
		response, errResp := http.Get(fmt.Sprintf("%s%s", s.GetHostPath(), "health"))
		if errResp != nil {
			log.Errorf("try %d, return health check of the rest service for error  %v", i, errResp)
			time.Sleep(healthCheckPause)
			err = errResp
			continue
		}
		if response != nil && response.StatusCode == http.StatusOK {
			response.Body.Close()
			log.Infof("rest service returned healthy status")
			time.Sleep(healthCheckPause)
			err = nil
			return
		}
		response.Body.Close()
	}
	if err != nil {
		err = fmt.Errorf("error connecting to rest api %s", err.Error())
	}
	return
}

// Port port id
func (s *Webhook) Port() int {
	return s.port
}

//Ready gives the status of the server
func (s *Webhook) Ready() bool {
	return s.status == started
}

// GetHostPath  returns hostpath
func (s *Webhook) GetHostPath() *types.URI {
	return types.ParseURI(fmt.Sprintf("http://localhost:%d%s", s.port, s.apiPath))
}

// Start will start res routes service
func (s *Webhook) Start() {
	if s.status == started || s.status == starting {
		log.Infof("Server is already running at port %d", s.port)
		return
	}
	s.status = starting
	r := mux.NewRouter()
	api := r.PathPrefix(s.apiPath).Subrouter()

	api.HandleFunc("/webhook", s.publishHwEvent).Methods(http.MethodPost)

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			log.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			log.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			log.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			log.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			log.Println("Methods:", strings.Join(methods, ","))
		}
		log.Println()
		return nil
	})

	if err != nil {
		log.Println(err)
	}
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, r)
	})

	log.Info("starting rest api server")
	log.Infof("endpoint %s", s.apiPath)
	go wait.Until(func() {
		s.status = started
		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.port),
			Handler: api,
		}
		err := s.httpServer.ListenAndServe()
		if err != nil {
			log.Errorf("restarting due to error with api server %s\n", err.Error())
			s.status = failed
		}
	}, 1*time.Second, s.closeCh)
}

// Shutdown ... shutdown rest service api, but it will not close until close chan is called
func (s *Webhook) Shutdown() {
	log.Warnf("trying to shutdown rest api sever, please use close channel to shutdown ")
	s.httpServer.Close()
}

// publishHwEvent gets redfish HW events and converts it to cloud native event and publishes to the hw publisher
func (s *Webhook) publishHwEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}

	data := HwEvent{}
	if err = json.Unmarshal(bodyBytes, &data); err != nil {
		respondWithError(w, err.Error())
		return
	}

	log.Printf("Received Webhook event data %v", data)
	log.Printf("To be sent to publisher ID %v", s.targetPub.ID)
}
