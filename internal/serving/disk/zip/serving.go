package zip

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/zip"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var instance = disk.New(vfs.Instrumented(zip.New("zip")), metrics.VFSServingFileSize)

// Instance returns a serving instance that is capable of reading files
// from a zip archives opened from a URL, most likely stored in object storage
func Instance() serving.Serving {
	return instance
}
