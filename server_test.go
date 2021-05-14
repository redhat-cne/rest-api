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

package restapi_test

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

	log "github.com/sirupsen/logrus"

	"github.com/redhat-cne/rest-api"
	"github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/types"
	api "github.com/redhat-cne/sdk-go/v1/pubsub"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var (
	server *restapi.Server

	eventOutCh chan *channel.DataChan
	closeCh    chan struct{}
	wg         sync.WaitGroup
	port       int    = 8989
	apPath     string = "/routes/cne/v1/"
	resource   string = "test/test"
	storePath  string = "."
	ObjSub     pubsub.PubSub
	ObjPub     pubsub.PubSub
)

func init() {
	eventOutCh = make(chan *channel.DataChan, 10)
	closeCh = make(chan struct{})
}

func TestMain(m *testing.M) {
	server = restapi.InitServer(port, apPath, storePath, eventOutCh, closeCh)
	//start http server
	server.Start()

	wg.Add(1)
	go func() {
		for d := range eventOutCh {
			log.Infof("incoming data %#v", d)
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
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "health"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_CreateSubscription(t *testing.T) {
	// create subscription
	sub := api.NewPubSub(
		&types.URI{URL: url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port), Path: fmt.Sprintf("%s%s", apPath, "dummy")}},
		resource)

	data, err := json.Marshal(&sub)
	assert.Nil(t, err)
	assert.NotNil(t, data)
	/// create new subscription
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "subscriptions"), bytes.NewBuffer(data))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	bodyString := string(bodyBytes)
	log.Print(bodyString)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	err = json.Unmarshal(bodyBytes, &ObjSub)
	assert.Nil(t, err)
	assert.NotEmpty(t, ObjSub.ID)
	assert.NotEmpty(t, ObjSub.URILocation)
	assert.NotEmpty(t, ObjSub.EndPointURI)
	assert.NotEmpty(t, ObjSub.Resource)
	assert.Equal(t, sub.Resource, ObjSub.Resource)
	log.Infof("Subscription:\n%s", ObjSub.String())
}

func TestServer_GetSubscription(t *testing.T) {
	// Get Just Created Subscription
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s%s/%s", port, apPath, "subscriptions", ObjSub.ID), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	rSub := api.New()
	err = json.Unmarshal(bodyBytes, &rSub)
	if e, ok := err.(*json.SyntaxError); ok {
		log.Infof("syntax error at byte offset %d", e.Offset)
	}
	assert.Nil(t, err)
	assert.Equal(t, rSub.ID, ObjSub.ID)
}

func TestServer_CreatePublisher(t *testing.T) {
	pub := pubsub.PubSub{
		ID:          "",
		EndPointURI: &types.URI{URL: url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port), Path: fmt.Sprintf("%s%s", apPath, "dummy")}},
		Resource:    resource,
	}
	pubData, err := json.Marshal(&pub)
	assert.Nil(t, err)
	assert.NotNil(t, pubData)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "publishers"), bytes.NewBuffer(pubData))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	pubBodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(pubBodyBytes, &ObjPub)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.NotEmpty(t, ObjPub.ID)
	assert.NotEmpty(t, ObjPub.URILocation)
	assert.NotEmpty(t, ObjPub.EndPointURI)
	assert.NotEmpty(t, ObjPub.Resource)
	assert.Equal(t, pub.Resource, ObjPub.Resource)
	log.Infof("publisher \n%s", ObjPub.String())
}

func TestServer_GetPublisher(t *testing.T) {
	// Get Just created Publisher
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s%s/%s", port, apPath, "publishers", ObjPub.ID), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	pubBodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	var rPub pubsub.PubSub
	log.Printf("the data %s", string(pubBodyBytes))
	err = json.Unmarshal(pubBodyBytes, &rPub)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Nil(t, err)
	assert.Equal(t, ObjPub.ID, rPub.ID)
}

func TestServer_ListSubscriptions(t *testing.T) {
	// Get All Subscriptions
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "subscriptions"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close() // Close body only if response non-nil
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	var subList []pubsub.PubSub
	log.Printf("TestServer_ListSubscriptions :%s\n", string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &subList)
	assert.Nil(t, err)
	assert.Greater(t, len(subList), 0)
}

func TestServer_ListPublishers(t *testing.T) {
	// Get All Publisher
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "publishers"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	pubBodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	var pubList []pubsub.PubSub
	err = json.Unmarshal(pubBodyBytes, &pubList)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Greater(t, len(pubList), 0)
}

func TestServer_TestPingStatusStatusCode(t *testing.T) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%d%s%s%s", port, apPath, "subscriptions/status/", ObjSub.ID), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestServer_DeleteSubscription(t *testing.T) {
	// Delete All Subscriptions
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "subscriptions"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
}

func TestServer_DeletePublisher(t *testing.T) {
	// Delete All Publisher
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "publishers"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
}

func TestServer_GetNonExistingPublisher(t *testing.T) {
	// Get Just created Publisher
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s%s/%s", port, apPath, "publishers", ObjPub.ID), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
	assert.Nil(t, err)
}

func TestServer_GetNonExistingSubscription(t *testing.T) {
	// Get Just Created Subscription
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d%s%s/%s", port, apPath, "subscriptions", ObjSub.ID), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func TestServer_TestDummyStatusCode(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "dummy"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestServer_KillAndRecover(t *testing.T) {
	server.Shutdown()
	time.Sleep(2*time.Second)
	// CHECK URL IS UP
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d%s%s", port, apPath, "health"), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.HTTPClient.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

}

func TestServer_End(t *testing.T) {
	close(eventOutCh)
	close(closeCh)
}
