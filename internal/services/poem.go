package services

import (
	"strings"

	"github.com/ssback/internal/models"
)

func ProcessPoem(p *models.Poem) {
	p.Text = strings.ReplaceAll(p.Text, `\n`, "\n")
	p.LineCount = len(strings.Split(p.Text, "\n"))
}

func ProcessPoems(poems []models.Poem) {
	for i := range poems {
		ProcessPoem(&poems[i])
	}
}
