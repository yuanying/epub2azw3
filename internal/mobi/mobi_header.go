package mobi

// languageCodeMap maps BCP 47 language tags to MOBI language codes.
var languageCodeMap = map[string]uint32{
	"en": 0x0409, // English (US)
	"ja": 0x0411, // Japanese
	"de": 0x0407, // German
	"fr": 0x040C, // French
	"es": 0x040A, // Spanish
	"it": 0x0410, // Italian
	"pt": 0x0416, // Portuguese (Brazil)
	"zh": 0x0804, // Chinese (Simplified)
	"ko": 0x0412, // Korean
	"nl": 0x0413, // Dutch
	"ru": 0x0419, // Russian
}

// defaultLanguageCode is used when the language is not found in the map.
const defaultLanguageCode = 0x0409

// LanguageCode converts a BCP 47 language tag to a MOBI language code.
// Returns defaultLanguageCode (English US) for unknown or empty strings.
func LanguageCode(lang string) uint32 {
	if code, ok := languageCodeMap[lang]; ok {
		return code
	}
	return defaultLanguageCode
}
