package markup

import "github.com/moriyoshi/ik"

type Writer interface {
	WriteString(s string) (int, error)
}

type MarkupRenderer interface {
    Render(markup *ik.Markup)
}
