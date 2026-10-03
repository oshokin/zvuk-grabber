package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlbum_UnmarshalPodcastEpisodePubDate(t *testing.T) {
	t.Parallel()

	const payload = `{
		"id": 43686140,
		"title": "Совсем другое дело",
		"type": "podcast",
		"metaType": "podcast",
		"artists": [],
		"volumes": [[
			{
				"id": "155836944",
				"title": "Вишневая девятка. Как я сдаю в аренду машину из девяностых",
				"pubDate": "2026-09-15",
				"artists": []
			}
		]]
	}`

	var album Album

	require.NoError(t, json.Unmarshal([]byte(payload), &album))

	require.Len(t, album.Volumes, 1)
	require.Len(t, album.Volumes[0], 1)
	assert.Equal(t, "podcast", album.Type)
	assert.Empty(t, album.Artists)
	assert.Equal(t, "2026-09-15", album.Volumes[0][0].PubDate)
}

func TestAlbum_UnmarshalAudiobookWithoutYear(t *testing.T) {
	t.Parallel()

	const payload = `{
		"id": 43888536,
		"title": "Татьяна Столяр. «Я есть жир»",
		"type": "audiobook",
		"metaType": "podcast",
		"artists": [{"id": 26582516, "name": "Татьяна Столяр"}],
		"volumes": [[
			{
				"id": "155603183",
				"title": "Татьяна Столяр. «Я есть жир». Часть 1",
				"artists": [{"id": 26582516, "name": "Татьяна Столяр"}]
			}
		]]
	}`

	var album Album

	require.NoError(t, json.Unmarshal([]byte(payload), &album))

	assert.Equal(t, 0, album.Year)
	assert.Empty(t, album.ReleaseDate)
	assert.Equal(t, "audiobook", album.Type)
	require.Len(t, album.Artists, 1)
	assert.Equal(t, "Татьяна Столяр", album.Artists[0].Name)
	require.NotEmpty(t, album.Volumes)
	require.NotEmpty(t, album.Volumes[0])
	assert.Empty(t, album.Volumes[0][0].PubDate)
}
