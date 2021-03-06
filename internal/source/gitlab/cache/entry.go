package cache

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Entry represents a cache object that can be retrieved asynchronously and
// holds a pointer to *api.Lookup when the domain lookup has been retrieved
// successfully
type Entry struct {
	domain                     string
	created                    time.Time
	refreshedOriginalTimestamp time.Time
	retrieve                   *sync.Once
	refresh                    *sync.Once
	mux                        *sync.RWMutex
	retrieved                  chan struct{}
	response                   *api.Lookup
	refreshTimeout             time.Duration
	expirationTimeout          time.Duration
	retriever                  *Retriever
}

func newCacheEntry(domain string, refreshTimeout, entryExpirationTimeout time.Duration, retriever *Retriever) *Entry {
	return &Entry{
		domain:            domain,
		created:           time.Now(),
		retrieve:          &sync.Once{},
		refresh:           &sync.Once{},
		mux:               &sync.RWMutex{},
		retrieved:         make(chan struct{}),
		refreshTimeout:    refreshTimeout,
		expirationTimeout: entryExpirationTimeout,
		retriever:         retriever,
	}
}

// IsUpToDate returns true if the entry has been resolved correctly and has not
// expired yet. False otherwise.
func (e *Entry) IsUpToDate() bool {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.isResolved() && !e.isOutdated()
}

// NeedsRefresh return true if the entry has been resolved correctly but it has
// expired since then.
func (e *Entry) NeedsRefresh() bool {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.isResolved() && e.isOutdated()
}

// Lookup returns a retriever Lookup response.
func (e *Entry) Lookup() *api.Lookup {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.response
}

// Retrieve perform a blocking retrieval of the cache entry response.
func (e *Entry) Retrieve(ctx context.Context) (lookup *api.Lookup) {
	// We run the code within an additional func() to run both `e.setResponse`
	// and `e.retrieve.Retrieve` asynchronously.
	e.retrieve.Do(func() { go func() { e.setResponse(e.retriever.Retrieve(e.domain)) }() })

	select {
	case <-ctx.Done():
		lookup = &api.Lookup{Name: e.domain, Error: errors.New("context done")}
	case <-e.retrieved:
		lookup = e.Lookup()
	}

	return lookup
}

// Refresh will update the entry in the store only when it gets resolved successfully.
// If an existing successful entry exists, it will only be replaced if the new resolved
// entry is successful too.
// Errored refreshed Entry responses will not replace the previously successful entry.response
// for a maximum time of e.expirationTimeout.
func (e *Entry) Refresh(store Store) {
	e.refresh.Do(func() {
		go e.refreshFunc(store)
	})
}

func (e *Entry) refreshFunc(store Store) {
	entry := newCacheEntry(e.domain, e.refreshTimeout, e.expirationTimeout, e.retriever)

	entry.Retrieve(context.Background())

	// do not replace existing Entry `e.response` when `entry.response` has an error
	// and `e` has not expired. See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/281.
	if !e.isExpired() && entry.hasTemporaryError() {
		entry.response = e.response
		entry.refreshedOriginalTimestamp = e.created
	}

	store.ReplaceOrCreate(e.domain, entry)
}

func (e *Entry) setResponse(lookup api.Lookup) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.response = &lookup
	close(e.retrieved)
}

func (e *Entry) isOutdated() bool {
	if !e.refreshedOriginalTimestamp.IsZero() {
		return time.Since(e.refreshedOriginalTimestamp) > e.refreshTimeout
	}

	return time.Since(e.created) > e.refreshTimeout
}

func (e *Entry) isResolved() bool {
	return e.response != nil
}

func (e *Entry) isExpired() bool {
	if !e.refreshedOriginalTimestamp.IsZero() {
		return time.Since(e.refreshedOriginalTimestamp) > e.expirationTimeout
	}

	return time.Since(e.created) > e.expirationTimeout
}

func (e *Entry) domainExists() bool {
	return !errors.Is(e.response.Error, domain.ErrDomainDoesNotExist)
}

// hasTemporaryError checks currently refreshed entry for errors after resolving the lookup again
// and is different to domain.ErrDomainDoesNotExist (this is an edge case to prevent serving
// a page right after being deleted).
func (e *Entry) hasTemporaryError() bool {
	if os.Getenv("FF_DISABLE_REFRESH_TEMPORARY_ERROR") == "true" {
		return false
	}

	return e.response != nil &&
		e.response.Error != nil &&
		e.domainExists()
}
