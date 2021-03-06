package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/local"
)

type customProjectResolver struct {
	config *domainConfig

	path string
}

func (p *customProjectResolver) Resolve(r *http.Request) (*serving.Request, error) {
	if p.config == nil {
		return nil, domain.ErrDomainDoesNotExist
	}

	lookupPath := &serving.LookupPath{
		ServingType:        "file",
		Prefix:             "/",
		Path:               p.path,
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ProjectID:          p.config.ID,
	}

	return &serving.Request{
		Serving:    local.Instance(),
		LookupPath: lookupPath,
		SubPath:    r.URL.Path,
	}, nil
}
