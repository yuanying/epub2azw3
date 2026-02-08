package epub

import (
	"testing"
)

func TestSplitFragment(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantPath     string
		wantFragment string
	}{
		{
			name:         "path with fragment",
			src:          "chapter1.xhtml#sec1",
			wantPath:     "chapter1.xhtml",
			wantFragment: "sec1",
		},
		{
			name:         "path without fragment",
			src:          "chapter1.xhtml",
			wantPath:     "chapter1.xhtml",
			wantFragment: "",
		},
		{
			name:         "fragment only",
			src:          "#sec1",
			wantPath:     "",
			wantFragment: "sec1",
		},
		{
			name:         "empty string",
			src:          "",
			wantPath:     "",
			wantFragment: "",
		},
		{
			name:         "multiple hash signs",
			src:          "chapter1.xhtml#sec1#subsec2",
			wantPath:     "chapter1.xhtml",
			wantFragment: "sec1#subsec2",
		},
		{
			name:         "path with directory",
			src:          "text/chapter1.xhtml#anchor",
			wantPath:     "text/chapter1.xhtml",
			wantFragment: "anchor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotFragment := splitFragment(tt.src)
			if gotPath != tt.wantPath {
				t.Errorf("splitFragment(%q) path = %q, want %q", tt.src, gotPath, tt.wantPath)
			}
			if gotFragment != tt.wantFragment {
				t.Errorf("splitFragment(%q) fragment = %q, want %q", tt.src, gotFragment, tt.wantFragment)
			}
		})
	}
}
