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

	// FailedDomainUpdates counts the number of failed site updates
	FailedDomainUpdates = prometheus.NewCounter(prometheus.CounterOpts{
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
)

func init() {
	prometheus.MustRegister(DomainsServed)
	prometheus.MustRegister(DomainUpdates)
	prometheus.MustRegister(DomainLastUpdateTime)
	prometheus.MustRegister(DomainsSourceCacheHit)
	prometheus.MustRegister(DomainsSourceCacheMiss)
}
