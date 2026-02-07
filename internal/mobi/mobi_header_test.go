package mobi

import "testing"

func TestLanguageCode(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want uint32
	}{
		{name: "Japanese", lang: "ja", want: 0x0411},
		{name: "English", lang: "en", want: 0x0409},
		{name: "German", lang: "de", want: 0x0407},
		{name: "French", lang: "fr", want: 0x040C},
		{name: "Unknown language defaults to English", lang: "zz", want: 0x0409},
		{name: "Empty string defaults to English", lang: "", want: 0x0409},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LanguageCode(tt.lang)
			if got != tt.want {
				t.Errorf("LanguageCode(%q) = 0x%04X, want 0x%04X", tt.lang, got, tt.want)
			}
		})
	}
}
