package localmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricStatus  ...
type MetricStatus string

const (
	// FAILCREATE ... failed to create subscription or publisher
	FAILCREATE MetricStatus = "failed to create"
	// FAILDELETE ... failed to delete subscription or publisher
	FAILDELETE MetricStatus = "failed to delete"
	// ACTIVE ...  active publishers and subscriptions
	ACTIVE MetricStatus = "active"
	// SUCCESS .... success events published
	SUCCESS MetricStatus = "success"
	// FAIL .... failed events published
	FAIL MetricStatus = "fail"
)

var (
	//eventPublishedCount ...  Total no of events published by the api
	eventPublishedCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_events_api_published",
			Help: "Metric to get number of events published by the rest api",
		}, []string{"address", "status"})

	//subscriptionCount ...  Total no of connection resets
	subscriptionCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_api_subscriptions",
			Help: "Metric to get number of subscriptions",
		}, []string{"status"})

	//publisherCount ...  Total no of events published by the transport
	publisherCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_api_publishers",
			Help: "Metric to get number of publishers",
		}, []string{"status"})
)

// RegisterMetrics ... register metrics
func RegisterMetrics() {
	prometheus.MustRegister(eventPublishedCount)
	prometheus.MustRegister(subscriptionCount)
	prometheus.MustRegister(publisherCount)
}

// UpdateEventPublishedCount ...
func UpdateEventPublishedCount(address string, status MetricStatus, val int) {
	eventPublishedCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}

// UpdateSubscriptionCount ...
func UpdateSubscriptionCount(status MetricStatus, val int) {
	subscriptionCount.With(
		prometheus.Labels{"status": string(status)}).Add(float64(val))
}

// UpdatePublisherCount ...
func UpdatePublisherCount(status MetricStatus, val int) {
	publisherCount.With(
		prometheus.Labels{"status": string(status)}).Add(float64(val))
}
