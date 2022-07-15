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

package publisher_api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redhat-cne/sdk-go/pkg/types"

	"github.com/redhat-cne/rest-api/pkg/localmetrics"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cne "github.com/redhat-cne/sdk-go/pkg/event"
	"github.com/redhat-cne/sdk-go/v1/event"

	"github.com/gorilla/mux"

	"github.com/redhat-cne/sdk-go/pkg/channel"

	"io/ioutil"
	"log"
	"net/http"
)

// publishEvent gets cloud native events and converts it to cloud event and publishes to a queue
//  to process by the consumer
func (s *Server) publishEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}
	cneEvent := event.CloudNativeEvent()
	if err = json.Unmarshal(bodyBytes, &cneEvent); err != nil {
		respondWithError(w, err.Error())
		return
	} // check if publisher is found
	pub, err := s.pubSubAPI.GetPublisher(cneEvent.ID)
	if err != nil {
		localmetrics.UpdateEventPublishedCount(cneEvent.ID, localmetrics.FAIL, 1)
		respondWithError(w, fmt.Sprintf("no publisher data for id %s found to publish event for", cneEvent.ID))
		return
	}
	ceEvent, err := cneEvent.NewCloudEvent(&pub)
	if err != nil {
		localmetrics.UpdateEventPublishedCount(pub.Resource, localmetrics.FAIL, 1)
		respondWithError(w, err.Error())
	} else {
		// publish event  to channel so sidecar will process and publish
		s.dataOut <- &channel.DataChan{
			Type:    channel.EVENT,
			Data:    ceEvent,
			Address: pub.GetResource(),
		}
		localmetrics.UpdateEventPublishedCount(pub.Resource, localmetrics.SUCCESS, 1)
		respondWithMessage(w, http.StatusAccepted, "Event sent")
	}
}

// pingForSubscribedEventStatus sends ping to the a listening address in the producer to fire all status as events
func (s *Server) pingForSubscribedEventStatus(w http.ResponseWriter, r *http.Request) {
	queries := mux.Vars(r)
	subscriptionID, ok := queries["subscriptionid"]
	if !ok {
		respondWithError(w, "subscription parameter not found")
		return
	}
	clientID, ok := queries["clientid"]
	if !ok {
		respondWithError(w, "clientid parameter not found")
		return
	}
	sub, err := s.pubSubAPI.GetSubscription(subscriptionID)
	if err != nil {
		respondWithError(w, "subscription not found")
		return
	}
	cneEvent := event.CloudNativeEvent()
	cneEvent.SetID(sub.ID)
	cneEvent.Type = "status_check"
	cneEvent.SetTime(types.Timestamp{Time: time.Now().UTC()}.Time)
	cneEvent.SetDataContentType(cloudevents.ApplicationJSON)
	cneEvent.SetData(cne.Data{
		Version: "v1",
	})
	//ceEvent, err := cneEvent.NewCloudEvent(&sub)

	if err != nil {
		respondWithError(w, err.Error())
	} else {
		// post event data to publisher end poit
		//TODO:
		//post to publisher service

		respondWithMessage(w, http.StatusAccepted, "ping sent")
	}
}

// logEvent gets cloud native events and converts it to cloud native event and writes to log
func (s *Server) logEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, err.Error())
		return
	}
	cneEvent := event.CloudNativeEvent()
	if err := json.Unmarshal(bodyBytes, &cneEvent); err != nil {
		respondWithError(w, err.Error())
		return
	} // check if publisher is found
	log.Printf("event received %v", cneEvent)
	respondWithMessage(w, http.StatusAccepted, "Event published to log")
}

func dummy(w http.ResponseWriter, r *http.Request) {
	respondWithMessage(w, http.StatusNoContent, "dummy test")
}

func respondWithError(w http.ResponseWriter, message string) {
	respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", cloudevents.ApplicationJSON)
	w.WriteHeader(code)
	w.Write(response) //nolint:errcheck
}
func respondWithMessage(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", cloudevents.ApplicationJSON)
	respondWithJSON(w, code, map[string]string{"status": message})
}

func respondWithByte(w http.ResponseWriter, code int, message []byte) {
	w.Header().Set("Content-Type", cloudevents.ApplicationJSON)
	w.WriteHeader(code)
	w.Write(message) //nolint:errcheck
}
