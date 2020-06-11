package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DomainsServed counts the total number of sites served
	DomainsServed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_served_domains",
		Help: "The number of sites served by this Pages app",
	})

	// DomainFailedUpdates counts the number of failed site updates
	DomainFailedUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_failed_total",
		Help: "The total number of site updates that have failed since daemon start",
	})

	// DomainUpdates counts the number of site updates successfully processed
	DomainUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_updated_total",
		Help: "The total number of site updates successfully processed since daemon start",
	})

	// DomainLastUpdateTime is the UNIX timestamp of the last update
	DomainLastUpdateTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_last_domain_update_seconds",
		Help: "UNIX timestamp of the last update",
	})

	// DomainsConfigurationUpdateDuration is the time it takes to update domains configuration from disk
	DomainsConfigurationUpdateDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_domains_configuration_update_duration",
		Help: "The time (in seconds) it takes to update domains configuration from disk",
	})

	// DomainsSourceCacheHit is the number of GitLab API call cache hits
	DomainsSourceCacheHit = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_cache_hit",
		Help: "The number of GitLab domains API cache hits",
	})

	// DomainsSourceCacheMiss is the number of GitLab API call cache misses
	DomainsSourceCacheMiss = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_cache_miss",
		Help: "The number of GitLab domains API cache misses",
	})

	// DomainsSourceFailures is the number of GitLab API calls that failed
	DomainsSourceFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_failures_total",
		Help: "The number of GitLab API calls that failed",
	})

	// ServerlessRequests measures the amount of serverless invocations
	ServerlessRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_serverless_requests",
		Help: "The number of total GitLab Serverless requests served",
	})

	// ServerlessLatency records serverless serving roundtrip duration
	ServerlessLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "gitlab_pages_serverless_latency",
		Help: "Serverless serving roundtrip duration",
	})

	// DomainsSourceAPIReqTotal is the number of calls made to the GitLab API that returned a 4XX error
	DomainsSourceAPIReqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_api_requests_total",
		Help: "The number of GitLab domains API calls with different status codes",
	}, []string{"status_code"})

	// DomainsSourceAPICallDuration is the time it takes to get a response from the GitLab API in seconds
	DomainsSourceAPICallDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gitlab_pages_domains_source_api_call_duration",
		Help: "The time (in seconds) it takes to get a response from the GitLab domains API",
	}, []string{"status_code"})

	// DiskServingFileSize metric for file size serving. serving_types: disk and object_storage
	DiskServingFileSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "gitlab_pages_disk_serving_file_size_bytes",
		Help: "The size in bytes for each file that has been served",
		// From 1B to 100MB in *10 increments (1 10 100 1,000 10,000 100,000 1'000,000 10'000,000 100'000,000)
		Buckets: prometheus.ExponentialBuckets(1.0, 10.0, 9),
	})

	// ServingTime metric for time taken to find a file serving it or not found.
	ServingTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gitlab_pages_serving_time_seconds",
		Help:    "The time (in seconds) taken to serve a file",
		Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 60, 180},
	})
)

// MustRegister collectors with the Prometheus client
func MustRegister() {
	prometheus.MustRegister(
		DomainsServed,
		DomainFailedUpdates,
		DomainUpdates,
		DomainLastUpdateTime,
		DomainsConfigurationUpdateDuration,
		DomainsSourceCacheHit,
		DomainsSourceCacheMiss,
		DomainsSourceAPIReqTotal,
		DomainsSourceAPICallDuration,
		DomainsSourceFailures,
		ServerlessRequests,
		ServerlessLatency,
		DiskServingFileSize,
		ServingTime,
	)
}
