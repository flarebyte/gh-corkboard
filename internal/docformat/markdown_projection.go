package docformat

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func projectMarkdown(source []byte) Projection {
	document := goldmark.DefaultParser().Parse(text.NewReader(source))
	headings := []any{}
	var offsets []int
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		heading, ok := node.(*ast.Heading)
		if !ok {
			continue
		}
		var visible strings.Builder
		appendInline(&visible, heading, source)
		headings = append(headings, map[string]any{
			"level": heading.Level,
			"text":  strings.Join(strings.Fields(visible.String()), " "),
		})
		offset := -1
		if heading.Lines().Len() != 0 {
			offset = heading.Lines().At(0).Start
		}
		offsets = append(offsets, offset)
	}
	return Projection{
		Value: map[string]any{"headings": headings},
		offset: func(path []string) int {
			if len(path) < 2 || path[0] != "headings" {
				return -1
			}
			i, err := strconv.Atoi(path[1])
			if err != nil || i < 0 || i >= len(offsets) {
				return -1
			}
			return offsets[i]
		},
	}
}

func appendInline(out *strings.Builder, parent ast.Node, source []byte) {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		switch inline := node.(type) {
		case *ast.Text:
			value := inline.Segment.Value(source)
			if !inline.IsRaw() {
				value = inlineText(value)
			}
			out.Write(value)
			if inline.HardLineBreak() || inline.SoftLineBreak() {
				out.WriteByte(' ')
			}
		case *ast.String:
			value := inline.Value
			if !inline.IsRaw() && !inline.IsCode() {
				value = inlineText(value)
			}
			out.Write(value)
		case *ast.AutoLink:
			out.Write(inline.Label(source))
		case *ast.CodeSpan:
			for child := inline.FirstChild(); child != nil; child = child.NextSibling() {
				value := child.(*ast.Text).Segment.Value(source)
				out.Write(value)
			}
		case *ast.RawHTML:
			// HTML tags themselves have no visible heading text.
		default:
			appendInline(out, node, source)
		}
	}
}

var inlineEscape = regexp.MustCompile(`\\[[:punct:]]|&(?:#[0-9]{1,7}|#[xX][0-9a-fA-F]{1,6}|[A-Za-z][A-Za-z0-9]*);`)

func inlineText(value []byte) []byte {
	return inlineEscape.ReplaceAllFunc(value, func(token []byte) []byte {
		if token[0] == '\\' {
			return token[1:]
		}
		if token[1] == '#' {
			return util.ResolveNumericReferences(token)
		}
		return util.ResolveEntityNames(token)
	})
}
