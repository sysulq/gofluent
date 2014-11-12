package markup

import (
	"html"
	"github.com/moriyoshi/ik"
)

type HTMLRenderer struct {
	Out Writer
}

var supportedAttrs = []int { ik.Embolden, ik.Underlined }

func (renderer *HTMLRenderer) tag(style int) string {
	if style == ik.Embolden {
		return "b"
	} else if style == ik.Underlined {
		return "u"
	} else {
		panic("never get here")
	}
}

func (renderer *HTMLRenderer) Render(markup *ik.Markup) {
	out := renderer.Out
	appliedAttrs := 0
	styleStack := make(ik.IntVector, 0)
	for _, chunk := range markup.Chunks {
		chunkAttrs := int(chunk.Attrs)
		removedAttrs := ^chunkAttrs & appliedAttrs
		for _, supportedAttr := range supportedAttrs {
			for removedAttrs & supportedAttr != 0 {
				poppedStyle := styleStack.Pop()
				out.WriteString("</")
				out.WriteString(renderer.tag(poppedStyle))
				out.WriteString(">")
				appliedAttrs &= ^poppedStyle
				removedAttrs &= ^poppedStyle
			}
		}
		newAttrs := chunkAttrs & ^appliedAttrs
		for _, supportedAttr := range supportedAttrs {
			if newAttrs & supportedAttr != 0 {
				styleStack.Append(supportedAttr)
				out.WriteString("<")
				out.WriteString(renderer.tag(supportedAttr))
				out.WriteString(">")
			}
		}
		appliedAttrs |= newAttrs
		out.WriteString(html.EscapeString(chunk.Text))
	}
	for len(styleStack) > 0 {
		poppedStyle := styleStack.Pop()
		out.WriteString("</")
		out.WriteString(renderer.tag(poppedStyle))
		out.WriteString(">")
	}
}
