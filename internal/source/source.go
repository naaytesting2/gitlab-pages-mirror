package source

import "gitlab.com/gitlab-org/gitlab-pages/internal/domain"

// Source represents an abstract interface of a domains configuration source.
type Source interface {
	GetDomain(string) (*domain.Domain, error)
	IsReady() bool
}
