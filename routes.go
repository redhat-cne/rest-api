package restapi

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	ce "github.com/cloudevents/sdk-go/v2/event"

	"github.com/redhat-cne/sdk-go/pkg/pubsub"

	"github.com/redhat-cne/sdk-go/v1/event"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/redhat-cne/sdk-go/pkg/channel"

	"io/ioutil"
	"log"
	"net/http"
)

//createSubscription create subscription and send it to a channel that is shared by middleware to process
// Creates a new subscription .
// If subscription exists with same resource then existing subscription is returned .
// responses:
//  201: repoResp
//  400: badReq
//  204: noContent
func (s *Server) createSubscription(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sub := pubsub.PubSub{}

	if err := json.Unmarshal(bodyBytes, &sub); err != nil {
		respondWithError(w, http.StatusBadRequest, "marshalling error")
		return
	}

	if sub.GetEndpointURI() != "" {
		response, err := s.HTTPClient.Post(sub.GetEndpointURI(), cloudevents.ApplicationJSON, nil)
		if err != nil {
			log.Printf("There was error validating endpointurl %v", err)
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			log.Printf("There was error validating endpointurl %s returned status code %d", sub.GetEndpointURI(), response.StatusCode)
			respondWithError(w, http.StatusBadRequest, "Return url validation check failed for create subscription.check endpointURI")
			return
		}
	}

	//check sub.EndpointURI by get
	sub.SetID(uuid.New().String())
	_ = sub.SetURILocation(fmt.Sprintf("http://localhost:%d%s%s/%s", s.port, s.apiPath, "subscriptions", sub.ID)) //noling:errcheck

	newSub, err := s.pubSubAPI.CreateSubscription(sub)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, newSub)
}

//createPublisher create publisher and send it to a channel that is shared by middleware to process
// Creates a new publisher .
// If publisher exists with same resource then existing publisher is returned .
// responses:
//  201: repoResp
//  400: badReq
//  204: noContent
func (s *Server) createPublisher(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pub := pubsub.PubSub{}

	if err := json.Unmarshal(bodyBytes, &pub); err != nil {
		respondWithError(w, http.StatusBadRequest, "marshalling error")
		return
	}

	if pub.GetEndpointURI() != "" {
		response, err := s.HTTPClient.Post(pub.GetEndpointURI(), cloudevents.ApplicationJSON, nil)
		if err != nil {
			log.Printf("There was error validating endpointurl %v", err)
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			log.Printf("There was error validating endpointurl %s returned status code %d", pub.GetEndpointURI(), response.StatusCode)
			respondWithError(w, http.StatusBadRequest, "Return url validation check failed for create publisher,check endpointURI")
			return
		}
	}

	//check sub.EndpointURI by get
	pub.SetID(uuid.New().String())
	_ = pub.SetURILocation(fmt.Sprintf("http://localhost:%d%s%s/%s", s.port, s.apiPath, "subscriptions", pub.ID)) //noling:errcheck

	newPub, err := s.pubSubAPI.CreatePublisher(pub)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	// go ahead and create QDR to this address
	s.sendOut(channel.SENDER, &newPub)
	respondWithJSON(w, http.StatusCreated, newPub)
}

func (s *Server) sendOut(eType channel.Type, sub *pubsub.PubSub) {
	// go ahead and create QDR to this address
	s.dataOut <- &channel.DataChan{
		Address: sub.GetResource(),
		Data:    &ce.Event{},
		Type:    eType,
		Status:  channel.NEW,
	}
}
func (s *Server) getSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	queries := mux.Vars(r)
	subscriptionID, ok := queries["subscriptionid"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "subscription not found")
		return
	}
	sub, err := s.pubSubAPI.GetSubscription(subscriptionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "subscription not found")
		return
	}
	respondWithJSON(w, http.StatusOK, sub)

}

func (s *Server) getPublisherByID(w http.ResponseWriter, r *http.Request) {
	queries := mux.Vars(r)
	publisherID, ok := queries["publisherid"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "publisher parameter is required")
		return
	}
	pub, err := s.pubSubAPI.GetPublisher(publisherID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "publisher not found")
		return
	}
	respondWithJSON(w, http.StatusOK, pub)

}
func (s *Server) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	b, err := s.pubSubAPI.GetSubscriptionsFromFile()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error loading subscriber data")
		return
	}
	respondWithByte(w, http.StatusOK, b)

}

func (s *Server) getPublishers(w http.ResponseWriter, r *http.Request) {
	b, err := s.pubSubAPI.GetPublishersFromFile()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error loading publishers data")
		return
	}
	respondWithByte(w, http.StatusOK, b)

}

func (s *Server) deletePublisher(w http.ResponseWriter, r *http.Request) {
	queries := mux.Vars(r)
	publisherID, ok := queries["publisherid"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "publisherid param is missing")
		return
	}
	if err := s.pubSubAPI.DeletePublisher(publisherID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithMessage(w, http.StatusOK, "OK")
}

func (s *Server) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	queries := mux.Vars(r)
	subscriptionID, ok := queries["subscriptionid"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "subscriptionid param is missing")
		return
	}
	if err := s.pubSubAPI.DeleteSubscription(subscriptionID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithMessage(w, http.StatusOK, "OK")
}
func (s *Server) deleteAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	if err := s.pubSubAPI.DeleteAllSubscriptions(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithMessage(w, http.StatusOK, "deleted all subscriptions")
}

func (s *Server) deleteAllPublishers(w http.ResponseWriter, r *http.Request) {
	if err := s.pubSubAPI.DeleteAllPublishers(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithMessage(w, http.StatusOK, "deleted all publishers")
}

//publishEvent gets cloud native events and converts it to cloud native event and publishes to a transport to send
//it to the consumer
func (s *Server) publishEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	cneEvent := event.CloudNativeEvent()
	if err := json.Unmarshal(bodyBytes, &cneEvent); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	} // check if publisher is found
	pub, err := s.pubSubAPI.GetPublisher(cneEvent.ID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No publisher data present to publish event")
		return
	}

	ceEvent, err := cneEvent.NewCloudEvent(&pub)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	} else {
		s.dataOut <- &channel.DataChan{
			Type:    channel.EVENT,
			Data:    ceEvent,
			Address: pub.GetResource(),
		}
		respondWithMessage(w, http.StatusAccepted, "Event sent")
	}
}

//logEvent gets cloud native events and converts it to cloud native event and writes to log
func (s *Server) logEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	cneEvent := event.CloudNativeEvent()
	if err := json.Unmarshal(bodyBytes, &cneEvent); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	} // check if publisher is found
	log.Printf("event received %v", cneEvent)
	respondWithMessage(w, http.StatusAccepted, "Event published to log")

}

func dummy(w http.ResponseWriter, r *http.Request) {
	respondWithMessage(w, http.StatusNoContent, "dummy test")
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", cloudevents.ApplicationJSON)
	respondWithJSON(w, code, map[string]string{"error": message})
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
