package restapi

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/v1/pubsub"
	"sync"

	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Server defines rest routes server object
type Server struct {
	port       int
	apiPath    string
	dataOut    chan<- *channel.DataChan
	close <-chan bool
	HTTPClient *http.Client
	pubSubAPI  *pubsub.API
}

// InitServer is used to supply configurations for rest routes server
func InitServer(port int, apiPath, storePath string, dataOut chan<- *channel.DataChan, close <-chan bool) *Server {
	server := Server{
		port:    port,
		apiPath: apiPath,
		dataOut: dataOut,
		close: close,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: 10 * time.Second,
		},
		pubSubAPI: pubsub.GetAPIInstance(storePath),
	}
	return &server
}

// Port port id
func (s *Server) Port() int {
	return s.port
}

//GetHostPath ...
func (s *Server) GetHostPath() string{
	return fmt.Sprintf("http://localhost:%d%s",s.port,s.apiPath)
}
// Start will start res routes service
func (s *Server) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	r := mux.NewRouter()
	api := r.PathPrefix(s.apiPath).Subrouter()

	//The POST method creates a subscription resource for the (Event) API consumer.
	// SubscriptionInfo  status 201
	// Shall be returned when the subscription resource created successfully.
	/*Request
	   {
		"ResourceType": "ptp",
	    "EndpointURI ": "http://localhost:9090/resourcestatus/ptp", /// daemon
		"ResourceQualifier": {
				"NodeName":"worker-1"
				"Source":"/cluster-x/worker-1/SYNC/ptp"
			}
		}
	Response:
			{
			//"SubscriptionID": "789be75d-7ac3-472e-bbbc-6d62878aad4a",
	        "PublisherId": "789be75d-7ac3-472e-bbbc-6d62878aad4a",
			"URILocation": "http://localhost:8080/ocloudNotifications/v1/subsciptions/789be75d-7ac3-472e-bbbc-6d62878aad4a",
			"ResourceType": "ptp",
	         "EndpointURI ": "http://localhost:9090/resourcestatus/ptp", // address where the event
				"ResourceQualifier": {
				"NodeName":"worker-1"
	              "Source":"/cluster-x/worker-1/SYNC/ptp"
			}
		}*/

	/*201 Shall be returned when the subscription resource created successfully.
		See note below.
	400 Bad request by the API consumer. For example, the endpoint URI does not include ‘localhost’.
	404 Subscription resource is not available. For example, ptp is not supported by the node.
	409 The subscription resource already exists.
	*/
	api.HandleFunc("/subscriptions", s.CreateSubscription).Methods(http.MethodPost)
	api.HandleFunc("/publishers", s.createPublisher).Methods(http.MethodPost)
	/*
		 this method a list of subscription object(s) and their associated properties
		200  Returns the subscription resources and their associated properties that already exist.
			See note below.
		404 Subscription resources are not available (not created).
	*/
	api.HandleFunc("/subscriptions", s.getSubscriptions).Methods(http.MethodGet)
	api.HandleFunc("/publishers", s.getPublishers).Methods(http.MethodGet)
	// 200 and 404
	api.HandleFunc("/subscriptions/{subscriptionid}", s.getSubscriptionByID).Methods(http.MethodGet)
	api.HandleFunc("/publishers/{publisherid}", s.getPublisherByID).Methods(http.MethodGet)
	// 204 on success or 404
	api.HandleFunc("/subscriptions/{subscriptionid}", s.deleteSubscription).Methods(http.MethodDelete)
	api.HandleFunc("/publishers/{publisherid}", s.deletePublisher).Methods(http.MethodDelete)

	api.HandleFunc("/subscriptions", s.deleteAllSubscriptions).Methods(http.MethodDelete)
	api.HandleFunc("/publishers", s.deleteAllPublishers).Methods(http.MethodDelete)

	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK") //nolint:errcheck
	}).Methods(http.MethodGet)

	api.HandleFunc("/dummy", dummy).Methods(http.MethodPost)
	api.HandleFunc("/log", s.logEvent).Methods(http.MethodPost)

	api.HandleFunc("/create/event", s.publishEvent).Methods(http.MethodPost)

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

	log.Print("Started Rest API Server")
	log.Printf("endpoint %s", s.apiPath)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), api))

}
