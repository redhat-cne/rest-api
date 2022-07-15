package subscription_api

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/util/wait"
	pubsubv1 "github.com/redhat-cne/sdk-go/v1/pubsub"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Package restapi Pub/Sub Rest API.
//
// Rest API spec .
//
// Terms Of Service:
//
//     Schemes: http, https
//     Host: k8Service
//     Version: 1.0.0
//     Contact: Aneesh Puttur<aputtur@redhat.com>
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//
// swagger:meta

import (
	"github.com/redhat-cne/sdk-go/pkg/event"
	"github.com/redhat-cne/sdk-go/pkg/types"
)

var once sync.Once

// SubscriptionInstance ... is singleton instance
var SubscriptionInstance *Server
var healthCheckPause time.Duration = 2 * time.Second

type serverStatus int

const (
	starting = iota
	started
	notReady
	failed
)

// Server defines rest routes server object
type Server struct {
	servicePort int
	serviceHost *string
	//data in from events
	dataOut    chan<- *channel.DataChan
	dataIn     <-chan *channel.DataChan
	HTTPClient *http.Client
	httpServer *http.Server
	pubSubAPI  *pubsubv1.API
	status     serverStatus
	closeCh    <-chan struct{}
}

// publisher/subscription data model
// swagger:response pubSubResp
type swaggPubSubRes struct { //nolint:deadcode,unused
	// in:body
	Body pubsub.PubSub
}

// PubSub request model
// swagger:response eventResp
type swaggPubSubEventRes struct { //nolint:deadcode,unused
	// in:body
	Body event.Event
}

// Error Bad Request
// swagger:response badReq
type swaggReqBadRequest struct { //nolint:deadcode,unused
	// in:body
	Body struct {
		// HTTP status code 400 -  Bad Request
		Code int `json:"code" example:"400"`
	}
}

// Error Not Found
// swagger:response notFoundReq
type swaggReqNotFound struct { //nolint:deadcode,unused
	// in:body
	Body struct {
		// HTTP status code 404 -  Not Found
		Code int `json:"code" example:"404"`
	}
}

// Accepted
// swagger:response acceptedReq
type swaggReqAccepted struct { //nolint:deadcode,unused
	// in:body
	Body struct {
		// HTTP status code 202 -  Accepted
		Code int `json:"code" example:"202"`
	}
}

// InitServer is used to supply configurations for rest routes server
func InitServer(port int, serviceHost, storePath string, dataIn <-chan *channel.DataChan, dataOut chan<- *channel.DataChan, closeCh <-chan struct{}) *Server {
	once.Do(func() {
		SubscriptionInstance = &Server{
			serviceHost: func(s string) *string {
				return &s
			}(serviceHost),
			servicePort: port,
			status:      notReady,
			dataIn:      dataIn,
			dataOut:     dataOut,
			closeCh:     closeCh,
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 20,
				},
				Timeout: 10 * time.Second,
			},
			pubSubAPI: pubsubv1.GetAPIInstance(storePath),
		}
	})
	// singleton
	return SubscriptionInstance
}

// EndPointHealthChk checks for rest service health
func (s *Server) EndPointHealthChk() (err error) {
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
func (s *Server) Port() int {
	return s.servicePort
}

//Ready gives the status of the server
func (s *Server) Ready() bool {
	return s.status == started
}

func (s *Server) GetHostPath() *types.URI {
	return types.ParseURI(fmt.Sprintf("http://%s:%d/", *s.serviceHost, s.servicePort))
}

// Start will start res routes service
func (s *Server) Start() {
	if s.status == started || s.status == starting {
		log.Infof("Server is already running at port %d", s.servicePort)
		return
	}
	s.status = starting
	r := mux.NewRouter()
	api := r.PathPrefix(s.GetHostPath().String()).Subrouter()

	api.HandleFunc("/status", s.pingForSubscribedEventStatus).Methods(http.MethodPut)

	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK") //nolint:errcheck
	}).Methods(http.MethodGet)

	api.HandleFunc("/dummy", dummy).Methods(http.MethodPost)
	api.HandleFunc("/log", s.logEvent).Methods(http.MethodPost)

	//publishEvent create event and send it to a channel that is shared by middleware to process
	// swagger:operation POST /publish event publishEvent
	// ---
	// summary: Creates a new event.
	// description: If publisher is present for the event, then event creation is success and be returned with Accepted (202).
	// parameters:
	// - name: event
	//   description: event along with publisher id
	//   in: body
	//   schema:
	//      "$ref": "#/definitions/Event"
	// responses:
	//   "202":
	//     "$ref": "#/responses/acceptedReq"
	//   "400":
	//     "$ref": "#/responses/badReq"
	api.HandleFunc("/event", s.publishEvent).Methods(http.MethodPost)

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

	log.Info("starting subscription service")
	log.Infof("endpoint %s", s.GetHostPath())
	go wait.Until(func() {
		s.status = started
		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", *s.serviceHost, s.servicePort),
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
func (s *Server) Shutdown() {
	log.Warnf("trying to shutdown rest api sever, please use close channel to shutdown ")
	s.httpServer.Close()
}
