package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseTemplateName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Author - Title", CollapseTemplateName("0000 - Author - Title"))
	assert.Equal(t, "Author - Title", CollapseTemplateName(" - Author - Title"))
	assert.Equal(t, "Совсем другое дело", CollapseTemplateName("Совсем другое дело - Совсем другое дело"))
	assert.Equal(t, "01 - Title", CollapseTemplateName("01 - Title"))
	assert.Equal(t, "Title", CollapseTemplateName("0000 - Title"))
	assert.Equal(
		t,
		"Татьяна Столяр. «Я есть жир»",
		CollapseTemplateName("Татьяна Столяр - Татьяна Столяр. «Я есть жир»"),
	)
	assert.Equal(
		t,
		"2026-09-15 - Вишневая девятка",
		CollapseTemplateName("2026-09-15 - Вишневая девятка"),
	)
}

func TestSanitizeTemplateName_DropsUnknownYear(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Безумная идея", SanitizeTemplateName("0000 - Безумная идея"))
}
