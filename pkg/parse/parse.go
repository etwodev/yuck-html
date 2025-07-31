package parse

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/net/html"
)

var log = zerolog.New(zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: "2006-01-02T15:04:05",
}).With().Timestamp().Str("service", "parse").Logger()

func TranspileYuckFromHTML(input string) (string, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	removeWhitespaceTextNodes(doc)

	var out strings.Builder
	out.WriteString(";; Auto-generated Yuck widgets from HTML ;;\n\n")
	var htmlNode *html.Node
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "html" {
			htmlNode = c
			break
		}
	}

	if htmlNode == nil {
		return "", fmt.Errorf("no <html> tag found in input")
	}

	var bodyNode *html.Node
	for c := htmlNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "body" {
			bodyNode = c
			break
		}
	}
	if bodyNode == nil {
		return "", fmt.Errorf("no <body> tag found inside <html>")
	}

	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {

		if c.Type != html.ElementNode {
			log.Warn().Msgf("Skipping non-element node: %s", c.Data)
			continue
		}

		// TODO: Handle type validation for attributes
		// var types []string
		var fields []string
		var name string
		for _, a := range c.Attr {
			if a.Key != "name" {
				fields = append(fields, a.Key)
			} else {
				name = a.Val
			}
		}

		switch c.Data {
		case "window":
			var geometry *html.Node
			var attrs []string

			for child := c.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && child.Data == "geometry" {
					geometry = child
				}
			}

			for _, a := range c.Attr {
				if a.Val == "" {
					attrs = append(attrs, fmt.Sprintf(":%s", a.Key))
				} else {
					attrs = append(attrs, fmt.Sprintf(":%s \"%s\"", a.Key, a.Val))
				}
			}

			if geometry != nil {
				var geomAttrs []string
				for _, ga := range geometry.Attr {
					geomAttrs = append(geomAttrs, fmt.Sprintf(":%s \"%s\"", ga.Key, ga.Val))
				}
				attrs = append(attrs, fmt.Sprintf(":geometry (geometry %s)", strings.Join(geomAttrs, "\n  ")))
			}

			out.WriteString(fmt.Sprintf("(defwindow %s\n  %s\n", name, strings.Join(attrs, "\n  ")))

			for child := c.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && child.Data == "geometry" {
					continue
				}
				walkWidgets(child, &out, 1, visitNode)
			}

			out.WriteString("\n\n)")

		case "widget":
			out.WriteString(fmt.Sprintf("(defwidget %s %s", name, fmt.Sprintf("[%s]", strings.Join(fields, " "))))
			walkWidgets(c.FirstChild, &out, 2, visitNode)
		default:
			log.Warn().Msgf("Unhandled top-level tag <%s>; skipping", c.Data)
			continue
		}
	}

	return out.String(), nil
}

func walkWidgets(n *html.Node, out *strings.Builder, level int, visit func(*html.Node, int) string) {
	if n == nil {
		out.WriteString(")")
		return
	}

	out.WriteString(visit(n, level)) // Write the current node
	if n.Type != html.TextNode {
		walkWidgets(n.FirstChild, out, level+1, visit) // Recur on the first child
	}
	walkWidgets(n.NextSibling, out, level, visit) // Recur on the next sibling
}

func visitNode(n *html.Node, level int) string {
	out := strings.Builder{}
	if n.Type == html.TextNode {
		return " " + getType(n.Data)
	}

	if n.Type != html.ElementNode {
		return ""
	}

	// TODO: Handle if the node is a valid widget type
	if n.Data == "" {
		return ""
	}

	// TODO: Handle if the attributes are valid for the given widget type
	var fields []string
	for _, a := range n.Attr {
		if a.Val == "" {
			continue
		}
		fields = append(fields, fmt.Sprintf(":%s %s", a.Key, getType(a.Val)))
	}

	prefix := indent(fmt.Sprintf("\n(%s", n.Data), level*2) + "\n"
	data := indent(strings.Join(fields, "\n"), (level+1)*2)

	out.WriteString(prefix + data)
	return out.String()
}

func indent(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func getType(attr string) string {
	if _, err := strconv.Atoi(attr); err == nil {
		return attr
	}

	if len(attr) > 1 && attr[0] == '{' && attr[len(attr)-1] == '}' {
		return attr
	}

	if len(attr) > 1 && attr[0] == '(' && attr[len(attr)-1] == ')' {
		return attr
	}

	if len(attr) > 1 && attr[0] == '[' && attr[len(attr)-1] == ']' {
		attr = attr[1 : len(attr)-1]
		return attr
	}

	if strings.ToLower(attr) == "true" || strings.ToLower(attr) == "false" {
		return strings.ToLower(attr)
	}

	return fmt.Sprintf("\"%s\"", attr)
}

func removeWhitespaceTextNodes(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.TextNode {
			trimmed := strings.TrimSpace(c.Data)
			if trimmed == "" {
				n.RemoveChild(c)
			} else {
				c.Data = trimmed
			}
		} else {
			removeWhitespaceTextNodes(c)
		}
		c = next
	}
}
