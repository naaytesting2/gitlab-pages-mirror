// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Modified copy of https://golang.org/src/crypto/x509/root_unix.go

package httptransport

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// certFileEnv is the environment variable which identifies where to locate
	// the SSL certificate file. If set this overrides the system default.
	certFileEnv = "SSL_CERT_FILE"

	// certDirEnv is the environment variable which identifies which directory
	// to check for SSL certificate files. If set this overrides the system default.
	// It is a colon separated list of directories.
	// See https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html.
	certDirEnv = "SSL_CERT_DIR"
)

func init() {
	// override and load SSL_CERT_FILE and SSL_CERT_DIR in OSX.
	loadExtraCerts = func() {
		if err := loadCertFile(); err != nil {
			log.WithError(err).Error("failed to read SSL_CERT_FILE")
		}

		if err := loadCertDir(); err != nil {
			log.WithError(err).Error("failed to load SSL_CERT_DIR")
		}
	}
}

func loadCertFile() error {
	sslCertFile := os.Getenv(certFileEnv)
	if sslCertFile == "" {
		return nil
	}

	data, err := ioutil.ReadFile(sslCertFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	sysPool.AppendCertsFromPEM(data)

	return nil
}

func loadCertDir() error {
	var firstErr error
	var dirs []string
	if d := os.Getenv(certDirEnv); d != "" {
		// OpenSSL and BoringSSL both use ":" as the SSL_CERT_DIR separator.
		// See:
		//  * https://golang.org/issue/35325
		//  * https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html
		dirs = strings.Split(d, ":")
	}

	for _, directory := range dirs {
		fis, err := readUniqueDirectoryEntries(directory)
		if err != nil {
			if firstErr == nil && !os.IsNotExist(err) {
				firstErr = err
			}
			continue
		}

		rootsAdded := false
		for _, fi := range fis {
			data, err := ioutil.ReadFile(directory + "/" + fi.Name())
			if err == nil && sysPool.AppendCertsFromPEM(data) {
				rootsAdded = true
			}
		}

		if rootsAdded {
			return nil
		}
	}

	return firstErr
}

// readUniqueDirectoryEntries is like ioutil.ReadDir but omits
// symlinks that point within the directory.
func readUniqueDirectoryEntries(dir string) ([]os.FileInfo, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	uniq := fis[:0]
	for _, fi := range fis {
		if !isSameDirSymlink(fi, dir) {
			uniq = append(uniq, fi)
		}
	}
	return uniq, nil
}

// isSameDirSymlink reports whether fi in dir is a symlink with a
// target not containing a slash.
func isSameDirSymlink(fi os.FileInfo, dir string) bool {
	if fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(filepath.Join(dir, fi.Name()))
	return err == nil && !strings.Contains(target, "/")
}
