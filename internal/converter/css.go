package converter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// pxValueRe matches px values for conversion to em.
var pxValueRe = regexp.MustCompile(`(\d+(?:\.\d+)?)px`)

// ptValueRe matches pt values for conversion to em.
var ptValueRe = regexp.MustCompile(`(\d+(?:\.\d+)?)pt`)

// declarationRe matches a CSS property-value pair.
var declarationRe = regexp.MustCompile(`(?i)^\s*([\w-]+)\s*:\s*(.*?)\s*;?\s*$`)

// negativeMarginRe matches negative numeric values in margin declarations.
var negativeMarginRe = regexp.MustCompile(`-\d`)

// TransformCSS removes forbidden CSS properties and converts units.
// It processes the CSS declaration by declaration, preserving structure.
// CSS comments and string literals are passed through without transformation.
func TransformCSS(css string) string {
	if css == "" {
		return ""
	}

	var result strings.Builder
	i := 0

	for i < len(css) {
		ch := css[i]

		// Handle CSS comments: pass through without processing
		if ch == '/' && i+1 < len(css) && css[i+1] == '*' {
			end := strings.Index(css[i+2:], "*/")
			if end == -1 {
				// Unterminated comment, pass through rest
				result.WriteString(css[i:])
				break
			}
			end += i + 2 + 2 // position after "*/"
			result.WriteString(css[i:end])
			i = end
			continue
		}

		// Pass through whitespace, selectors, and braces
		if ch == '{' || ch == '}' {
			result.WriteByte(ch)
			i++
			continue
		}

		// Inside a declaration block, find individual declarations
		// Look for property: value; patterns
		if ch == ';' {
			result.WriteByte(ch)
			i++
			continue
		}

		// Try to find a declaration (property: value;)
		declEnd := findDeclarationEnd(css, i)
		if declEnd > i {
			decl := css[i:declEnd]

			// Check if this contains a colon (it's a declaration)
			if m := declarationRe.FindStringSubmatch(strings.TrimSpace(decl)); m != nil {
				property := m[1]
				value := m[2]

				if isForbiddenProperty(property, value) {
					// Skip this declaration but preserve the semicolon
					i = declEnd
					// Skip trailing semicolon and whitespace
					for i < len(css) && (css[i] == ';' || css[i] == ' ' || css[i] == '\t') {
						if css[i] == ';' {
							i++
							break
						}
						i++
					}
					continue
				}

				// Convert units and output
				converted := convertUnits(decl)
				result.WriteString(converted)
				i = declEnd
				continue
			}
		}

		// Pass through anything else (selectors, comments, etc.)
		result.WriteByte(ch)
		i++
	}

	return result.String()
}

// findDeclarationEnd finds the end of a CSS declaration starting at pos.
// Returns the position after the declaration (before or at the semicolon).
// It correctly handles string literals inside values (e.g., content: "...").
func findDeclarationEnd(css string, pos int) int {
	for i := pos; i < len(css); i++ {
		switch css[i] {
		case ';':
			return i
		case '{', '}':
			return i
		case '"', '\'':
			// Skip string literal
			quote := css[i]
			i++
			for i < len(css) {
				if css[i] == '\\' {
					i++ // skip escaped char
				} else if css[i] == quote {
					break
				}
				i++
			}
		}
	}
	return len(css)
}

// isForbiddenProperty checks if a CSS property-value pair should be removed.
func isForbiddenProperty(property, value string) bool {
	propertyLower := strings.ToLower(strings.TrimSpace(property))
	valueLower := strings.ToLower(strings.TrimSpace(value))

	switch propertyLower {
	case "position":
		return valueLower == "fixed" || valueLower == "absolute"
	case "transform":
		return true
	case "transition":
		return true
	}

	// transition-* properties (transition-property, transition-duration, etc.)
	if strings.HasPrefix(propertyLower, "transition-") {
		return true
	}

	// animation and animation-* properties
	if propertyLower == "animation" || strings.HasPrefix(propertyLower, "animation-") {
		return true
	}

	// Negative margins
	if propertyLower == "margin" || strings.HasPrefix(propertyLower, "margin-") {
		if strings.Contains(valueLower, "-") {
			if negativeMarginRe.MatchString(valueLower) {
				return true
			}
		}
	}

	return false
}

// convertUnits converts px and pt values to em in a CSS string fragment.
func convertUnits(s string) string {
	// Convert px to em (÷16)
	s = pxValueRe.ReplaceAllStringFunc(s, func(match string) string {
		submatch := pxValueRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		val, err := strconv.ParseFloat(submatch[1], 64)
		if err != nil {
			return match
		}
		return formatEm(val / 16.0)
	})

	// Convert pt to em (÷12)
	s = ptValueRe.ReplaceAllStringFunc(s, func(match string) string {
		submatch := ptValueRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		val, err := strconv.ParseFloat(submatch[1], 64)
		if err != nil {
			return match
		}
		return formatEm(val / 12.0)
	})

	return s
}

// formatEm formats an em value, omitting unnecessary decimal places.
func formatEm(val float64) string {
	if val == float64(int(val)) {
		return fmt.Sprintf("%dem", int(val))
	}
	s := strconv.FormatFloat(val, 'f', -1, 64)
	return s + "em"
}
