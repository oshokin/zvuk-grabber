package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseTemplateName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"0000 - Author - Title", "Author - Title"},
		{" - Author - Title", "Author - Title"},
		{"Совсем другое дело - Совсем другое дело", "Совсем другое дело"},
		{"01 - Title", "01 - Title"},
		{"0000 - Title", "Title"},
		{"Татьяна Столяр - Татьяна Столяр. «Я есть жир»", "Татьяна Столяр. «Я есть жир»"},
		{"2026-09-15 - Вишневая девятка", "2026-09-15 - Вишневая девятка"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, CollapseTemplateName(tc.in), tc.in)
	}
}

func TestSanitizeTemplateName_DropsUnknownYear(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Безумная идея", SanitizeTemplateName("0000 - Безумная идея"))
}
