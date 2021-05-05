---
Title: Metrics
---

REST-API populates [Prometheus][prometheus] collectors for metrics reporting. The metrics can be used for real-time monitoring and debugging.
rest-api metrics collector does not persist its metrics; if a member restarts, the metrics will be reset.

The simplest way to see the available metrics is to cURL the metrics endpoint `/metrics`. The format is described [here](http://prometheus.io/docs/instrumenting/exposition_formats/).

Follow the [Prometheus getting started doc](http://prometheus.io/docs/introduction/getting_started/) to spin up a Prometheus server to collect metrics.

The naming of metrics follows the suggested [Prometheus best practices](http://prometheus.io/docs/practices/naming/).

A metric name has an `cne`  prefix as its namespace, and a subsystem prefix .

###Registering collector in your application
The collector needs to be registered in the consuming application by calling `RegisterMetrics()`  method from `rest-api/pkg/localmetrics package`


## cne namespace metrics

The metrics under the `cne` prefix are for monitoring .  If there is any change of these metrics, it will be included in release notes.


### Metrics

These metrics describe the status of the cloud native events, publisher and subscriptions .

All these metrics are prefixed with `cne_`

| Name                                                  | Description                                              | Type    |
|-------------------------------------------------------|----------------------------------------------------------|---------|
| cne_events_api_published          | Metric to get number of events published by the rest api.   | Gauge |
| cne_api_subscriptions     | Metric to get number of subscriptions.  | Gauge   |
| cne_api_publishers     | Metric to get number of publishers.  | Gauge   |


`cne_events_api_published` -  The number of events published via rest-api, and their status by address.

Example
```json 
# HELP cne_events_api_published Metric to get number of events published by the rest api
# TYPE cne_events_api_published gauge
cne_events_api_published{address="/news-service/finance",status="success"} 9
cne_events_api_published{address="/news-service/sports",status="success"} 9
```

`cne_api_subscriptions` -  This metrics indicates number of subscriptions that are active.

Example
```json
# HELP cne_api_subscriptions Metric to get number of subscriptions
# TYPE cne_api_subscriptions gauge
cne_api_subscriptions{status="active"} 2
```

`cne_api_publishers` -  This metrics indicates number of publishers that are active.

Example
```json
# HELP cne_api_publishers Metric to get number of publishers
# TYPE cne_api_publishers gauge
cne_api_publishers{status="active"} 2
```



