// +build !go1.14,!go1.15

package par1

import (
	"embed"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/stretchr/testify/require"
)

//go:embed testdata
var testdataFS embed.FS

func TestCreateGolden(t *testing.T) {
	fileData := make(map[string][]byte)
	parData := make(map[string][]byte)
	entries, err := testdataFS.ReadDir("testdata")
	require.NoError(t, err)
	var parPath string
	for _, entry := range entries {
		name := entry.Name()
		data, err := testdataFS.ReadFile("testdata/" + name)
		require.NoError(t, err)
		ext := filepath.Ext(name)
		if strings.HasPrefix(ext, ".p") {
			if ext == ".par" {
				parPath = name
			}
			parData[name] = data
		} else {
			fileData[name] = data
		}
	}
	require.True(t, len(fileData) > 0)
	require.True(t, len(parPath) > 0)
	require.True(t, len(parData) > 0)

	fs := memfs.MakeMemFS(memfs.RootDir(), fileData)

	sortedPaths := fs.Paths()
	sort.Strings(sortedPaths)

	err = create(testFileIO{t, fs}, parPath, sortedPaths, CreateOptions{})
	require.NoError(t, err)

	for name, expectedData := range parData {
		data, err := fs.ReadFile(name)
		require.NoError(t, err)
		require.Equal(t, expectedData, data)
	}
}
