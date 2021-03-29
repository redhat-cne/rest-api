module github.com/redhat-cne/rest-api

go 1.15

require (
	github.com/cloudevents/sdk-go/v2 v2.3.1-0.20210302080936-5c462007a5d5
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/redhat-cne/sdk-go v0.0.0-20210328201601-fd9ed3a4e18c
	github.com/stretchr/testify v1.7.0
)

//replace github.com/redhat-cne/sdk-go => /home/aputtur/github.com/redhat-cne/sdk-go
