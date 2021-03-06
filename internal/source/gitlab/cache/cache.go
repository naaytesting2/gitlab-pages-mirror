package cache

import (
	"context"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	client api.Client
	store  Store
}

// NewCache creates a new instance of Cache.
func NewCache(client api.Client, cc *config.Cache) *Cache {
	return &Cache{
		client: client,
		store:  newMemStore(client, cc),
	}
}

// Resolve is going to return a lookup based on a domain name. The caching
// algorithm works as follows:
// - We first check if the cache entry exists, and if it is up-to-date. If it
//   is fresh we return the lookup entry from cache and it is a cache hit.
// - If entry is not up-to-date, that means it has been created in a cache
//   more than `entryRefreshTimeout` duration ago,  we schedule an asynchronous
//   retrieval of the latest configuration we are going to obtain through the
//   API, and we immediately return an old value, to avoid blocking clients. In
//   this case it is also a cache hit.
// - If cache entry has not been populated with a lookup information yet, we
//   block all the clients and make them wait until we retrieve the lookup from
//   the GitLab API. Clients should not wait for longer than
//   `retrievalTimeout`. It is a cache miss.
//
// We are going to retrieve a lookup from GitLab API using a retriever type. In
// case of failures (when GitLab API client returns an error) we will retry the
// operation a few times, waiting `maxRetrievalInterval` in between, total
// amount of requests is defined as `maxRetrievalRetries`. In case of an
// erroneous response, we will cache it, and it get recycled as every other
// cache entry.
//
// Examples:
// 1. Everything works
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - we successfully retrieve a lookup
//  - we cache this response
//  - and we pass it upstream to all clients
// 2. A domain does not exist
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - GitLab responded with a lookup and 204 HTTP status
//  - we cache this response with domain being `nil`
//  - we pass this lookup upstream to all the clients
// 3. GitLab is not responding
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - GitLab does not respond or responds with an error
//  - we retry this retrieval every `maxRetrievalInterval`
//  - we retry this retrieval `maxRetrievalRetries` in total
//  - we create a lookup that contains information about an error
//  - we cache this response
//  - we pass this lookup upstream to all the clients
func (c *Cache) Resolve(ctx context.Context, domain string) *api.Lookup {
	entry := c.store.LoadOrCreate(domain)

	if entry.IsUpToDate() {
		metrics.DomainsSourceCacheHit.Inc()
		return entry.Lookup()
	}

	if entry.NeedsRefresh() {
		entry.Refresh(c.store)

		metrics.DomainsSourceCacheHit.Inc()
		return entry.Lookup()
	}

	metrics.DomainsSourceCacheMiss.Inc()
	return entry.Retrieve(ctx)
}

// Status calls the client Status to check connectivity with the API
func (c *Cache) Status() error {
	return c.client.Status()
}
