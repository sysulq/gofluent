package markup

import (
	"strings"
	"github.com/moriyoshi/ik"
)

type TerminalEscapeRenderer struct {
	Out Writer
}

func (renderer *TerminalEscapeRenderer) Render(markup *ik.Markup) {
	out := renderer.Out
	appliedAttrs := 0
	_codes := [4]string {}
	for _, chunk := range markup.Chunks {
		codes := _codes[:0]
		chunkAttrs := int(chunk.Attrs)
		removedAttrs := ^chunkAttrs & appliedAttrs
		if chunkAttrs & ik.White != 0 {
			appliedAttrs &= ^ik.White
			removedAttrs |= ik.White
		}
		newAttrs := chunkAttrs
		if removedAttrs != 0 {
			codes = append(codes, "0")
		}
		if newAttrs & ik.Embolden != 0 {
			codes = append(codes, "1")
		}
		if newAttrs & ik.Underlined != 0 {
			codes = append(codes, "4")
		}
		if newAttrs & ik.White != 0 {
			codes = append(codes, "3031323334353637"[(newAttrs & ik.White) * 2:][:2])
		}
		if len(codes) > 0 {
			out.WriteString("\x1b[" + strings.Join(codes, ";") + "m")
		}
		appliedAttrs |= newAttrs
		out.WriteString(chunk.Text)
	}
	if appliedAttrs != 0 {
		out.WriteString("\x1b[0m")
	}
}
