package gitlab

import (
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/local"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/zip"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// fabricateLookupPath fabricates a serving LookupPath based on the API LookupPath
// `size` argument is DEPRECATED, see
// https://gitlab.com/gitlab-org/gitlab-pages/issues/272
func fabricateLookupPath(size int, lookup api.LookupPath) *serving.LookupPath {
	return &serving.LookupPath{
		ServingType:        lookup.Source.Type,
		Path:               lookup.Source.Path,
		Prefix:             lookup.Prefix,
		IsNamespaceProject: (lookup.Prefix == "/" && size > 1),
		IsHTTPSOnly:        lookup.HTTPSOnly,
		HasAccessControl:   lookup.AccessControl,
		ProjectID:          uint64(lookup.ProjectID),
	}
}

// fabricateServing fabricates serving based on the GitLab API response
func fabricateServing(lookup api.LookupPath) serving.Serving {
	source := lookup.Source

	switch source.Type {
	case "file":
		return local.Instance()
	case "zip":
		return zip.Instance()
	case "serverless":
		log.Errorf("attempted to fabricate serverless serving for project %d", lookup.ProjectID)

		// This feature has been disalbed, for more details see
		//   https://gitlab.com/gitlab-org/gitlab-pages/-/issues/467
		//
		// serving, err := serverless.NewFromAPISource(source.Serverless)
		// if err != nil {
		// 	log.WithError(err).Errorf("could not fabricate serving for project %d", lookup.ProjectID)
		//
		// 	break
		// }
		//
		// return serving
	}

	return defaultServing()
}

func defaultServing() serving.Serving {
	return local.Instance()
}
