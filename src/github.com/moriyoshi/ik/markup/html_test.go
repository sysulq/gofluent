package markup

import (
	"testing"
	"bytes"
	"github.com/moriyoshi/ik"
)

func TestHTMLRenderer_Render1(t *testing.T) {
	out := bytes.Buffer {}
	renderer := &HTMLRenderer { &out }
	renderer.Render(&ik.Markup {
		[]ik.MarkupChunk {
			{ 0, "test" },
			{ ik.Embolden, "EMBOLDEN" },
			{ ik.Underlined, "_underlined_" },
		},
	})
	if out.String() != "test<b>EMBOLDEN</b><u>_underlined_</u>" {
		t.Fail()
	}
}

func TestHTMLRenderer_Render2(t *testing.T) {
	out := bytes.Buffer {}
	renderer := &HTMLRenderer { &out }
	renderer.Render(&ik.Markup {
		[]ik.MarkupChunk {
			{ 0, "test" },
			{ ik.Embolden, "EMBOLDEN" },
			{ ik.Embolden | ik.Underlined, "_UNDERLINED_" },
			{ ik.Underlined, "_underlined_" },
		},
	})
	if out.String() != "test<b>EMBOLDEN<u>_UNDERLINED_</u></b><u>_underlined_</u>" {
		t.Fail()
	}
}
