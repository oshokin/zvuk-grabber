package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClassify_MixedProvidersAndUnknown verifies URL classification across providers and unknown inputs.
func TestClassify_MixedProvidersAndUnknown(t *testing.T) {
	t.Parallel()

	classified := Classify([]string{
		"https://zvuk.com/release/1",
		"https://music.yandex.ru/album/2",
		"https://example.com/other",
	})

	assert.Equal(t, []string{"https://zvuk.com/release/1"}, classified.Zvuk)
	assert.Equal(t, []string{"https://music.yandex.ru/album/2"}, classified.Yandex)
	assert.Equal(t, []string{"https://example.com/other"}, classified.Unknown)
}

// TestFlatten_ExpandsTextFileAndDeduplicates verifies Flatten expands text files and removes duplicate URLs.
func TestFlatten_ExpandsTextFileAndDeduplicates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	urlListPath := filepath.Join(tempDir, "urls.txt")
	content := `
# comment
https://zvuk.com/release/1
https://music.yandex.ru/album/2

https://music.yandex.ru/album/2
`
	err := os.WriteFile(urlListPath, []byte(content), 0o600)
	require.NoError(t, err)

	flattened, err := Flatten([]string{urlListPath, "https://zvuk.com/release/1", "https://example.com/item"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://zvuk.com/release/1",
		"https://music.yandex.ru/album/2",
		"https://example.com/item",
	}, flattened)
}
