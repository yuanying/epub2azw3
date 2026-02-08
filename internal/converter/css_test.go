package converter

import (
	"strings"
	"testing"
)

func TestTransformCSS_PositionFixedRemoved(t *testing.T) {
	css := `div { position: fixed; color: red; }`
	result := TransformCSS(css)
	if strings.Contains(result, "position") {
		t.Fatalf("position: fixed should be removed, got: %s", result)
	}
	if !strings.Contains(result, "color: red") {
		t.Fatalf("color: red should be preserved, got: %s", result)
	}
}

func TestTransformCSS_PositionAbsoluteRemoved(t *testing.T) {
	css := `div { position: absolute; }`
	result := TransformCSS(css)
	if strings.Contains(result, "position") {
		t.Fatalf("position: absolute should be removed, got: %s", result)
	}
}

func TestTransformCSS_PositionRelativePreserved(t *testing.T) {
	css := `div { position: relative; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "position: relative") {
		t.Fatalf("position: relative should be preserved, got: %s", result)
	}
}

func TestTransformCSS_PositionStaticPreserved(t *testing.T) {
	css := `div { position: static; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "position: static") {
		t.Fatalf("position: static should be preserved, got: %s", result)
	}
}

func TestTransformCSS_TransformRemoved(t *testing.T) {
	css := `div { transform: rotate(45deg); color: blue; }`
	result := TransformCSS(css)
	if strings.Contains(result, "transform") {
		t.Fatalf("transform should be removed, got: %s", result)
	}
	if !strings.Contains(result, "color: blue") {
		t.Fatalf("color: blue should be preserved, got: %s", result)
	}
}

func TestTransformCSS_TransitionRemoved(t *testing.T) {
	css := `div { transition: all 0.3s ease; }`
	result := TransformCSS(css)
	if strings.Contains(result, "transition") {
		t.Fatalf("transition should be removed, got: %s", result)
	}
}

func TestTransformCSS_AnimationRemoved(t *testing.T) {
	tests := []struct {
		name string
		css  string
	}{
		{"animation", `div { animation: fade 1s; }`},
		{"animation-name", `div { animation-name: fade; }`},
		{"animation-duration", `div { animation-duration: 1s; }`},
		{"animation-delay", `div { animation-delay: 0.5s; }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformCSS(tt.css)
			if strings.Contains(result, "animation") {
				t.Fatalf("%s should be removed, got: %s", tt.name, result)
			}
		})
	}
}

func TestTransformCSS_NegativeMarginRemoved(t *testing.T) {
	css := `div { margin-left: -10px; margin-right: 20px; }`
	result := TransformCSS(css)
	if strings.Contains(result, "margin-left") {
		t.Fatalf("negative margin-left should be removed, got: %s", result)
	}
	if !strings.Contains(result, "margin-right") {
		t.Fatalf("positive margin-right should be preserved, got: %s", result)
	}
}

func TestTransformCSS_NegativeMarginShorthand(t *testing.T) {
	css := `div { margin: -5px 10px; }`
	result := TransformCSS(css)
	if strings.Contains(result, "margin") {
		t.Fatalf("margin with negative value should be removed, got: %s", result)
	}
}

func TestTransformCSS_PositiveMarginPreserved(t *testing.T) {
	css := `div { margin: 10px; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "margin") {
		t.Fatalf("positive margin should be preserved, got: %s", result)
	}
}

func TestTransformCSS_WritingModePreserved(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want string
	}{
		{"writing-mode", `div { writing-mode: vertical-rl; }`, "writing-mode"},
		{"-epub-writing-mode", `div { -epub-writing-mode: vertical-rl; }`, "-epub-writing-mode"},
		{"-webkit-writing-mode", `div { -webkit-writing-mode: vertical-rl; }`, "-webkit-writing-mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformCSS(tt.css)
			if !strings.Contains(result, tt.want) {
				t.Fatalf("%s should be preserved, got: %s", tt.want, result)
			}
		})
	}
}

func TestTransformCSS_TextOrientationPreserved(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want string
	}{
		{"text-orientation", `div { text-orientation: mixed; }`, "text-orientation"},
		{"text-combine-upright", `div { text-combine-upright: all; }`, "text-combine-upright"},
		{"-webkit-text-combine", `div { -webkit-text-combine: horizontal; }`, "-webkit-text-combine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformCSS(tt.css)
			if !strings.Contains(result, tt.want) {
				t.Fatalf("%s should be preserved, got: %s", tt.want, result)
			}
		})
	}
}

func TestTransformCSS_TextEmphasisPreserved(t *testing.T) {
	css := `div { text-emphasis-style: dot; text-emphasis-position: over right; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "text-emphasis-style") {
		t.Fatalf("text-emphasis-style should be preserved, got: %s", result)
	}
	if !strings.Contains(result, "text-emphasis-position") {
		t.Fatalf("text-emphasis-position should be preserved, got: %s", result)
	}
}

func TestTransformCSS_RubyPositionPreserved(t *testing.T) {
	css := `ruby { ruby-position: over; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "ruby-position") {
		t.Fatalf("ruby-position should be preserved, got: %s", result)
	}
}

func TestTransformCSS_PxToEm(t *testing.T) {
	css := `div { font-size: 16px; margin: 32px; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "1em") {
		t.Fatalf("16px should be converted to 1em, got: %s", result)
	}
	if !strings.Contains(result, "2em") {
		t.Fatalf("32px should be converted to 2em, got: %s", result)
	}
}

func TestTransformCSS_PtToEm(t *testing.T) {
	css := `div { font-size: 12pt; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "1em") {
		t.Fatalf("12pt should be converted to 1em, got: %s", result)
	}
}

func TestTransformCSS_PercentPreserved(t *testing.T) {
	css := `div { width: 50%; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "50%") {
		t.Fatalf("percentage should be preserved, got: %s", result)
	}
}

func TestTransformCSS_EmPreserved(t *testing.T) {
	css := `div { font-size: 1.5em; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "1.5em") {
		t.Fatalf("em value should be preserved, got: %s", result)
	}
}

func TestTransformCSS_RemPreserved(t *testing.T) {
	css := `div { font-size: 2rem; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "2rem") {
		t.Fatalf("rem value should be preserved, got: %s", result)
	}
}

func TestTransformCSS_TransitionSubPropertiesRemoved(t *testing.T) {
	tests := []struct {
		name string
		css  string
	}{
		{"transition-property", `div { transition-property: opacity; }`},
		{"transition-duration", `div { transition-duration: 0.3s; }`},
		{"transition-timing-function", `div { transition-timing-function: ease; }`},
		{"transition-delay", `div { transition-delay: 0.1s; }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformCSS(tt.css)
			if strings.Contains(result, "transition") {
				t.Fatalf("%s should be removed, got: %s", tt.name, result)
			}
		})
	}
}

func TestTransformCSS_CSSCommentIgnored(t *testing.T) {
	css := `div { /* position: fixed; */ color: red; }`
	result := TransformCSS(css)
	if !strings.Contains(result, "position: fixed") {
		t.Fatalf("position: fixed inside comment should be preserved, got: %s", result)
	}
	if !strings.Contains(result, "color: red") {
		t.Fatalf("color: red should be preserved, got: %s", result)
	}
}

func TestTransformCSS_CSSStringLiteralIgnored(t *testing.T) {
	css := `div::before { content: "position: fixed"; color: blue; }`
	result := TransformCSS(css)
	if !strings.Contains(result, `"position: fixed"`) {
		t.Fatalf("string literal should be preserved, got: %s", result)
	}
	if !strings.Contains(result, "color: blue") {
		t.Fatalf("color: blue should be preserved, got: %s", result)
	}
}

func TestTransformCSS_CSSCommentMultiline(t *testing.T) {
	css := `
/* transform: rotate(45deg);
   animation: fade 1s;
*/
div { color: green; }
`
	result := TransformCSS(css)
	if !strings.Contains(result, "transform: rotate(45deg)") {
		t.Fatalf("transform inside comment should be preserved, got: %s", result)
	}
	if !strings.Contains(result, "color: green") {
		t.Fatalf("color: green should be preserved, got: %s", result)
	}
}

func TestTransformCSS_Empty(t *testing.T) {
	result := TransformCSS("")
	if result != "" {
		t.Fatalf("empty CSS should return empty, got: %q", result)
	}
}

func TestTransformCSS_MultipleDeclarations(t *testing.T) {
	css := `
.chapter {
  position: fixed;
  writing-mode: vertical-rl;
  transform: scale(1.5);
  font-size: 16px;
  margin-left: -20px;
  color: #333;
}`
	result := TransformCSS(css)
	if strings.Contains(result, "position") {
		t.Fatal("position: fixed should be removed")
	}
	if !strings.Contains(result, "writing-mode: vertical-rl") {
		t.Fatal("writing-mode should be preserved")
	}
	if strings.Contains(result, "transform") {
		t.Fatal("transform should be removed")
	}
	if !strings.Contains(result, "1em") {
		t.Fatal("16px should be converted to 1em")
	}
	if strings.Contains(result, "margin-left") {
		t.Fatal("negative margin should be removed")
	}
	if !strings.Contains(result, "color: #333") {
		t.Fatal("color should be preserved")
	}
}
