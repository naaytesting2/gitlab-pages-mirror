package redirects

import (
	"context"
	"io/ioutil"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func generateRedirectsFile(dirPath string, count int) error {
	content := strings.Repeat("/goto.html /target.html 301\n", count)
	content = content + "/entrance.html /exit.html 301\n"

	return ioutil.WriteFile(path.Join(dirPath, ConfigFile), []byte(content), 0600)
}

func benchmarkRedirectsRewrite(b *testing.B, redirectsCount int) {
	ctx := context.Background()

	root, tmpDir, cleanup := testhelpers.TmpDir(nil, "ParseRedirects_benchmarks")
	defer cleanup()

	err := generateRedirectsFile(tmpDir, redirectsCount)
	require.NoError(b, err)

	url, err := url.Parse("/entrance.html")
	require.NoError(b, err)

	redirects := ParseRedirects(ctx, root)
	require.NoError(b, redirects.error)

	for i := 0; i < b.N; i++ {
		_, _, err := redirects.Rewrite(url)
		require.NoError(b, err)
	}
}

func BenchmarkRedirectsRewrite(b *testing.B) {
	b.Run("10 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 10) })
	b.Run("100 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 100) })
	b.Run("1000 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 1000) })
}

func benchmarkRedirectsParseRedirects(b *testing.B, redirectsCount int) {
	ctx := context.Background()

	root, tmpDir, cleanup := testhelpers.TmpDir(nil, "ParseRedirects_benchmarks")
	defer cleanup()

	err := generateRedirectsFile(tmpDir, redirectsCount)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		redirects := ParseRedirects(ctx, root)
		require.NoError(b, redirects.error)
	}
}

func BenchmarkRedirectsParseRedirects(b *testing.B) {
	b.Run("10 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 10) })
	b.Run("100 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 100) })
	b.Run("1000 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 1000) })
}
