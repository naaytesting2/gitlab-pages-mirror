package disk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/redirects"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric *prometheus.HistogramVec
	vfs            vfs.VFS
}

// Show the user some validation messages for their _redirects file
func (reader *Reader) serveRedirectsStatus(w http.ResponseWriter, redirects *redirects.Redirects) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	_, err := fmt.Fprintln(w, redirects.Status())
	return err
}

func (reader *Reader) tryRedirects(w http.ResponseWriter, r *http.Request, lookupPath *serving.LookupPath) error {
	ctx := r.Context()
	root, err := reader.vfs.Root(ctx, lookupPath.Path)
	if err != nil {
		return err
	}

	redirs := redirects.ParseRedirects(ctx, root)

	rewrittenURL, status, err := redirs.Rewrite(r.URL)
	if err != nil {
		return err
	}

	http.Redirect(w, r, rewrittenURL.Path, status)

	return nil
}

func (reader *Reader) tryFile(w http.ResponseWriter, r *http.Request, lookupPath *serving.LookupPath) error {
	ctx := r.Context()

	root, err := reader.vfs.Root(ctx, lookupPath.Path)
	if err != nil {
		return err
	}

	fullPath, err := reader.resolvePath(ctx, root, lookupPath.SubPath)

	// request := h.Request
	host := r.Host
	urlPath := r.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			fullPath, err = reader.resolvePath(ctx, root, lookupPath.SubPath,
				"index.html")
		} else {
			// TODO why are we doing that? In tests it redirects to HTTPS. This seems wrong,
			// issue about this: https://gitlab.com/gitlab-org/gitlab-pages/issues/273

			// Concat Host with URL.Path
			redirectPath := "//" + host + "/"
			redirectPath += strings.TrimPrefix(urlPath, "/")

			// Ensure that there's always "/" at end
			redirectPath = strings.TrimSuffix(redirectPath, "/") + "/"
			http.Redirect(w, r, redirectPath, 302)
			return nil
		}
	}

	if locationError, _ := err.(*locationFileNoExtensionError); locationError != nil {
		fullPath, err = reader.resolvePath(ctx, root,
			strings.TrimSuffix(lookupPath.SubPath, "/")+".html")
	}

	if err != nil {
		return err
	}

	// Serve status of `_redirects` under `_redirects`
	// We check if the final resolved path is `_redirects` after symlink traversal
	if fullPath == redirects.ConfigFile {
		if os.Getenv("FF_ENABLE_REDIRECTS") != "false" {
			r := redirects.ParseRedirects(ctx, root)
			return reader.serveRedirectsStatus(w, r)
		}

		w.WriteHeader(http.StatusForbidden)
		return nil
	}

	return reader.serveFile(ctx, w, r, root, fullPath, lookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(w http.ResponseWriter, r *http.Request, lookupPath *serving.LookupPath) error {
	ctx := r.Context()

	root, err := reader.vfs.Root(ctx, lookupPath.Path)
	if err != nil {
		return err
	}

	page404, err := reader.resolvePath(ctx, root, "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(ctx, w, r, http.StatusNotFound,
		root, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(ctx context.Context, root vfs.Root, subPath ...string) (string, error) {
	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := strings.Join(subPath, "/")
	fullPath, err := symlink.EvalSymlinks(ctx, root, testPath)

	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return "", err
	}

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: testPath,
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (reader *Reader) serveFile(ctx context.Context, w http.ResponseWriter, r *http.Request, root vfs.Root, origPath string, accessControl bool) error {
	fullPath := reader.handleContentEncoding(ctx, w, r, root, origPath)

	file, err := root.Open(ctx, fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := reader.detectContentType(ctx, root, origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)

	reader.fileSizeMetric.WithLabelValues(reader.vfs.Name()).Observe(float64(fi.Size()))

	// Support vfs.SeekableFile if available (uncompressed files)
	if rs, ok := file.(vfs.SeekableFile); ok {
		http.ServeContent(w, r, origPath, fi.ModTime(), rs)
	} else {
		// compressed files will be served by io.Copy
		// TODO: Add extra headers https://gitlab.com/gitlab-org/gitlab-pages/-/issues/466
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		io.Copy(w, file)
	}

	return nil
}

func (reader *Reader) serveCustomFile(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, root vfs.Root, origPath string) error {
	fullPath := reader.handleContentEncoding(ctx, w, r, root, origPath)

	// Open and serve content of file
	file, err := root.Open(ctx, fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	contentType, err := reader.detectContentType(ctx, root, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.WithLabelValues(reader.vfs.Name()).Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}
