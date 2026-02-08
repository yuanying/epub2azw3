package converter

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func parseTestHTML(html string) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		panic(err)
	}
	return doc
}

func TestTransformHTML_ArticleToDiv(t *testing.T) {
	doc := parseTestHTML(`<html><body><article>content</article></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("div.article")
	if sel.Length() == 0 {
		t.Fatal("expected <article> to be converted to <div class=\"article\">")
	}
	if sel.Text() != "content" {
		t.Fatalf("content mismatch: got %q", sel.Text())
	}
}

func TestTransformHTML_SectionToDiv(t *testing.T) {
	doc := parseTestHTML(`<html><body><section>content</section></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("div.section")
	if sel.Length() == 0 {
		t.Fatal("expected <section> to be converted to <div class=\"section\">")
	}
}

func TestTransformHTML_AsideNavHeaderFooterFigure(t *testing.T) {
	tags := []string{"aside", "nav", "header", "footer", "figure"}
	for _, tag := range tags {
		t.Run(tag, func(t *testing.T) {
			doc := parseTestHTML(`<html><body><` + tag + `>content</` + tag + `></body></html>`)
			TransformHTML(doc)
			sel := doc.Find("div." + tag)
			if sel.Length() == 0 {
				t.Fatalf("expected <%s> to be converted to <div class=%q>", tag, tag)
			}
		})
	}
}

func TestTransformHTML_FigcaptionToP(t *testing.T) {
	doc := parseTestHTML(`<html><body><figcaption>caption text</figcaption></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("p.figcaption")
	if sel.Length() == 0 {
		t.Fatal("expected <figcaption> to be converted to <p class=\"figcaption\">")
	}
	if sel.Text() != "caption text" {
		t.Fatalf("content mismatch: got %q", sel.Text())
	}
}

func TestTransformHTML_ClassMerge(t *testing.T) {
	doc := parseTestHTML(`<html><body><article class="intro">content</article></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("div")
	if sel.Length() == 0 {
		t.Fatal("expected <article> to be converted to <div>")
	}
	class, _ := sel.Attr("class")
	if !strings.Contains(class, "intro") || !strings.Contains(class, "article") {
		t.Fatalf("expected class to contain both 'intro' and 'article', got %q", class)
	}
}

func TestTransformHTML_DataAttributeRemoval(t *testing.T) {
	doc := parseTestHTML(`<html><body><div data-custom="value" data-id="123" id="test">content</div></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("#test")
	if sel.Length() == 0 {
		t.Fatal("expected div to exist")
	}
	if _, exists := sel.Attr("data-custom"); exists {
		t.Fatal("data-custom attribute should be removed")
	}
	if _, exists := sel.Attr("data-id"); exists {
		t.Fatal("data-id attribute should be removed")
	}
	if _, exists := sel.Attr("id"); !exists {
		t.Fatal("id attribute should be preserved")
	}
}

func TestTransformHTML_ForbiddenAttributeRemoval(t *testing.T) {
	doc := parseTestHTML(`<html><body><div contenteditable="true" draggable="true" hidden="" spellcheck="false" translate="no" id="test">content</div></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("#test")
	for _, attr := range []string{"contenteditable", "draggable", "hidden", "spellcheck", "translate"} {
		if _, exists := sel.Attr(attr); exists {
			t.Fatalf("%s attribute should be removed", attr)
		}
	}
}

func TestTransformHTML_PreservedAttributes(t *testing.T) {
	doc := parseTestHTML(`<html><body><div class="test" id="myid" lang="ja" dir="rtl" style="color:red">content</div></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("#myid")
	for _, attr := range []string{"class", "id", "lang", "dir", "style"} {
		if _, exists := sel.Attr(attr); !exists {
			t.Fatalf("%s attribute should be preserved", attr)
		}
	}
}

func TestTransformHTML_RubyPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><ruby>漢<rt>かん</rt></ruby></body></html>`)
	TransformHTML(doc)
	// Ruby elements should be untouched
	if doc.Find("ruby").Length() == 0 {
		t.Fatal("ruby element should be preserved")
	}
	if doc.Find("rt").Length() == 0 {
		t.Fatal("rt element should be preserved")
	}
}

func TestTransformHTML_RpPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><ruby>漢<rp>(</rp><rt>かん</rt><rp>)</rp></ruby></body></html>`)
	TransformHTML(doc)
	if doc.Find("rp").Length() != 2 {
		t.Fatalf("expected 2 rp elements, got %d", doc.Find("rp").Length())
	}
}

func TestTransformHTML_SpanTcyPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><span class="tcy">12</span></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("span.tcy")
	if sel.Length() == 0 {
		t.Fatal("span.tcy should be preserved")
	}
	if sel.Text() != "12" {
		t.Fatalf("span.tcy content mismatch: got %q", sel.Text())
	}
}

func TestTransformHTML_SpanUprPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><span class="upr">ABC</span></body></html>`)
	TransformHTML(doc)
	sel := doc.Find("span.upr")
	if sel.Length() == 0 {
		t.Fatal("span.upr should be preserved")
	}
}

func TestTransformHTML_NestedTagConversion(t *testing.T) {
	doc := parseTestHTML(`<html><body><article><section><p>text</p></section></article></body></html>`)
	TransformHTML(doc)
	// article -> div.article, section -> div.section
	html, _ := doc.Find("body").Html()
	if !strings.Contains(html, "div") {
		t.Fatal("expected nested tags to be converted to divs")
	}
	if doc.Find("div.article").Length() == 0 {
		t.Fatal("expected div.article")
	}
	if doc.Find("div.section").Length() == 0 {
		t.Fatal("expected div.section")
	}
	if doc.Find("p").Length() == 0 {
		t.Fatal("p should be preserved")
	}
}

func TestTransformHTML_HrefPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><a href="http://example.com">link</a></body></html>`)
	TransformHTML(doc)
	href, exists := doc.Find("a").Attr("href")
	if !exists || href != "http://example.com" {
		t.Fatalf("href should be preserved, got %q", href)
	}
}

func TestTransformHTML_SrcPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><img src="image.jpg"/></body></html>`)
	TransformHTML(doc)
	src, exists := doc.Find("img").Attr("src")
	if !exists || src != "image.jpg" {
		t.Fatalf("src should be preserved, got %q", src)
	}
}

func TestTransformHTML_XmlLangPreserved(t *testing.T) {
	doc := parseTestHTML(`<html><body><p xml:lang="ja">text</p></body></html>`)
	TransformHTML(doc)
	lang, exists := doc.Find("p").Attr("xml:lang")
	if !exists || lang != "ja" {
		t.Fatalf("xml:lang should be preserved, got %q", lang)
	}
}
