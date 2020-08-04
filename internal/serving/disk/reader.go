package disk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/gocloud"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric prometheus.Histogram

	_vfs    vfs.VFS
	vfsOnce sync.Once
}

func (reader *Reader) vfs() vfs.VFS {
	reader.vfsOnce.Do(func() {
		if url := os.Getenv("GITLAB_PAGES_BUCKET_URL"); url != "" {
			var err error
			reader._vfs, err = gocloud.New(context.Background(), url, "")
			if err != nil {
				panic(err)
			}
		} else {
			reader._vfs = local.VFS{Root: "."}
		}
	})

	return reader._vfs
}

func (reader *Reader) tryFile(h serving.Handler) error {
	ctx := h.Request.Context()
	fullPath, err := reader.resolvePath(ctx, h.LookupPath.Path, h.SubPath)

	request := h.Request
	host := request.Host
	urlPath := request.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			fullPath, err = reader.resolvePath(ctx, h.LookupPath.Path, h.SubPath, "index.html")
		} else {
			// TODO why are we doing that? In tests it redirects to HTTPS. This seems wrong,
			// issue about this: https://gitlab.com/gitlab-org/gitlab-pages/issues/273

			// Concat Host with URL.Path
			redirectPath := "//" + host + "/"
			redirectPath += strings.TrimPrefix(urlPath, "/")

			// Ensure that there's always "/" at end
			redirectPath = strings.TrimSuffix(redirectPath, "/") + "/"
			http.Redirect(h.Writer, h.Request, redirectPath, 302)
			return nil
		}
	}

	if locationError, _ := err.(*locationFileNoExtensionError); locationError != nil {
		fullPath, err = reader.resolvePath(ctx, h.LookupPath.Path, strings.TrimSuffix(h.SubPath, "/")+".html")
	}

	if err != nil {
		return err
	}

	return reader.serveFile(h.Writer, h.Request, fullPath, h.LookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(h serving.Handler) error {
	page404, err := reader.resolvePath(h.Request.Context(), h.LookupPath.Path, "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(h.Writer, h.Request, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(ctx context.Context, publicPath string, subPath ...string) (string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + strings.Join(subPath, "/")
	fullPath, err := symlink.EvalSymlinks(ctx, reader.vfs(), testPath)

	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return "", err
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, publicPath) && fullPath != filepath.Clean(publicPath) {
		return "", fmt.Errorf("%q should be in %q", fullPath, publicPath)
	}

	fi, err := reader.vfs().Lstat(ctx, fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: strings.TrimPrefix(fullPath, publicPath),
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (reader *Reader) serveFile(w http.ResponseWriter, r *http.Request, origPath string, accessControl bool) error {
	fullPath := reader.handleGZip(w, r, origPath)

	ctx := r.Context()
	file, err := reader.vfs().Open(ctx, fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := reader.vfs().Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := reader.detectContentType(ctx, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (reader *Reader) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
	ctx := r.Context()
	fullPath := reader.handleGZip(w, r, origPath)

	// Open and serve content of file
	file, err := reader.vfs().Open(ctx, fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := reader.vfs().Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	contentType, err := reader.detectContentType(ctx, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}
