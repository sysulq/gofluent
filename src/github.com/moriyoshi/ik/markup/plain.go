package markup

import (
	"github.com/moriyoshi/ik"
)

type PlainRenderer struct {
	Out Writer
}

func (renderer *PlainRenderer) Render(markup *ik.Markup) {
	for _, chunk := range markup.Chunks {
		renderer.Out.WriteString(chunk.Text)
	}
}
