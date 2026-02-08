package converter

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// tagConversions maps HTML5 semantic tags to their Kindle-compatible replacements.
var tagConversions = map[string]string{
	"article":    "div",
	"section":    "div",
	"aside":      "div",
	"nav":        "div",
	"header":     "div",
	"footer":     "div",
	"figure":     "div",
	"figcaption": "p",
}

// forbiddenAttrs lists attributes that should be removed from all elements.
var forbiddenAttrs = map[string]bool{
	"contenteditable": true,
	"draggable":       true,
	"hidden":          true,
	"spellcheck":      true,
	"translate":       true,
}

// TransformHTML transforms HTML5 tags to Kindle-compatible equivalents
// and removes forbidden attributes.
func TransformHTML(doc *goquery.Document) {
	// Convert HTML5 semantic tags
	for origTag, newTag := range tagConversions {
		doc.Find(origTag).Each(func(i int, s *goquery.Selection) {
			existingClass, _ := s.Attr("class")
			if existingClass != "" {
				s.SetAttr("class", existingClass+" "+origTag)
			} else {
				s.SetAttr("class", origTag)
			}
			// Change the tag name by manipulating the underlying node
			s.Get(0).Data = newTag
		})
	}

	// Remove forbidden attributes and data-* attributes from all elements
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		node := s.Get(0)
		var toRemove []string
		for _, attr := range node.Attr {
			if forbiddenAttrs[attr.Key] || strings.HasPrefix(attr.Key, "data-") {
				toRemove = append(toRemove, attr.Key)
			}
		}
		for _, key := range toRemove {
			s.RemoveAttr(key)
		}
	})
}
