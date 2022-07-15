// Copyright 2020 The Cloud Native Events Authors
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

package publisher_api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	event "github.com/redhat-cne/sdk-go/pkg/event"
	log "github.com/sirupsen/logrus"

	"github.com/redhat-cne/rest-api/subscription-api"
	"github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var (
	server *subscription_api.Server

	eventOutCh  chan *channel.DataChan
	eventInCh   chan *channel.DataChan
	closeCh     chan struct{}
	wg          sync.WaitGroup
	port        int    = 8787
	serviceHost string = "localhost"
	resource    string = "test/test"
	storePath   string = "../"
	ObjSub      pubsub.PubSub
	ObjPub      pubsub.PubSub
)

func init() {
	eventOutCh = make(chan *channel.DataChan, 10)
	eventInCh = make(chan *channel.DataChan, 10)
	closeCh = make(chan struct{})
}

func TestMain(m *testing.M) {
	server = subscription_api.InitServer(port, serviceHost, storePath, eventInCh, eventOutCh, closeCh)
	//start http server
	server.Start()

	wg.Add(1)
	go func() {
		for d := range eventInCh {
			log.Infof("run by http protocol: data received to create subscriptions with producer %#v", d)
			//calls producer post to create client
		}
	}()
	wg.Add(1)
	go func() {
		for d := range eventOutCh {
			log.Infof("run by main sidecar: event recieved to post to client data %#v", d)
		}
	}()
	time.Sleep(2 * time.Second)
	port = server.Port()
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestServer_Health(t *testing.T) {
	// CHECK URL IS UP
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d/%s", serviceHost, port, "health"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_TestDummyStatusCode(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://%s:%d/%s", serviceHost, port, "dummy"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestServer_KillAndRecover(t *testing.T) {
	server.Shutdown()
	time.Sleep(2 * time.Second)
	// CHECK URL IS UP
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d/%s", serviceHost, port, "health"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

}

// New get new rest client
func NewRestClient() *Rest {
	return &Rest{
		client: http.Client{
			Timeout: 1 * time.Second,
		},
	}
}

func publishEvent(e event.Event) {
	//create publisher
	url := &types.URI{URL: url.URL{Scheme: "http",
		Host: fmt.Sprintf("localhost:%d", port),
		Path: fmt.Sprintf("http://%s:%d/%s", serviceHost, port, "event")}}
	rc := NewRestClient()
	err := rc.PostEvent(url, e)
	if err != nil {
		log.Errorf("error publishing events %v to url %s", err, url.String())
	} else {
		log.Debugf("published event %s", e.ID)
	}
}

func TestServer_End(t *testing.T) {
	close(eventOutCh)
	close(eventInCh)
	close(closeCh)
}

// Rest client to make http request
type Rest struct {
	client http.Client
}

// Post post with data
func (r *Rest) Post(url *types.URI, data []byte) int {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, "POST", url.String(), bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("error creating post request %v", err)
		return http.StatusBadRequest
	}
	request.Header.Set("content-type", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		log.Errorf("error in post response %v", err)
		return http.StatusBadRequest
	}
	if response.Body != nil {
		defer response.Body.Close()
		// read any content and print
		body, readErr := ioutil.ReadAll(response.Body)
		if readErr == nil && len(body) > 0 {
			log.Debugf("%s return response %s\n", url.String(), string(body))
		}
	}
	return response.StatusCode
}

// New get new rest client
func New() *Rest {
	return &Rest{
		client: http.Client{
			Timeout: 1 * time.Second,
		},
	}
}

// PostEvent post an event to the give url and check for error
func (r *Rest) PostEvent(url *types.URI, e event.Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		log.Errorf("error marshalling event %v", e)
		return err
	}
	if status := r.Post(url, b); status == http.StatusBadRequest {
		return fmt.Errorf("post returned status %d", status)
	}
	return nil
}
