package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

func day(unix int64) string { return time.Unix(unix, 0).Format("2006-01-02") }

func fileName(m Msg, ext string) string {
	name := fmt.Sprintf("%s %s - %s", day(m.Date), m.FromName, m.Subject)
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return -1
		}
		return r
	}, name)
	if r := []rune(name); len(r) > 150 {
		name = string(r[:150])
	}
	return strings.TrimRight(strings.TrimSpace(name), ".") + ext
}

// saveEmail writes m to saveDir as self-contained HTML (images inlined) or Markdown.
func saveEmail(dataDir, saveDir string, m Msg, format string) (string, error) {
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return "", err
	}
	var out, ext string
	switch format {
	case "html":
		ext = ".html"
		out = rewrite(m.HTML, func(u string) string {
			b, ct, err := fetchImage(dataDir, u)
			if err != nil {
				return u
			}
			return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(b)
		})
		out = strings.Replace(out, "<head>", "<head><meta charset=\"utf-8\"><title>"+html.EscapeString(m.Subject)+"</title>", 1)
	case "md":
		ext = ".md"
		body, err := htmltomarkdown.ConvertString(m.HTML)
		if err != nil {
			return "", err
		}
		out = fmt.Sprintf("# %s\n\n*%s <%s>, %s*\n\n%s\n", m.Subject, m.FromName, m.FromAddr, day(m.Date), body)
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
	path := filepath.Join(saveDir, fileName(m, ext))
	return path, os.WriteFile(path, []byte(out), 0o644)
}

// highlightsMarkdown renders highlights (already ordered by issue) grouped under one heading per issue.
func highlightsMarkdown(hs []Highlight) string {
	var b strings.Builder
	b.WriteString("# Highlights\n")
	var cur int64 = -1
	for _, h := range hs {
		if h.MessageID != cur {
			cur = h.MessageID
			fmt.Fprintf(&b, "\n## %s\n\n*%s, %s*\n", h.Subject, h.FromName, day(h.Date))
		}
		b.WriteString("\n> " + strings.ReplaceAll(strings.TrimSpace(h.Text), "\n", "\n> ") + "\n")
		if n := strings.TrimSpace(h.Note); n != "" {
			b.WriteString("\n" + n + "\n")
		}
	}
	return b.String()
}
