package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const fixture = "From: Morning Brew <crew@MorningBrew.com>\r\n" +
	"Subject: =?utf-8?q?Caf=C3=A9_markets?=\r\n" +
	"Date: Mon, 21 Sep 2026 07:00:00 +0000\r\n" +
	"Message-ID: <abc@mb>\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/alternative; boundary=XX\r\n\r\n" +
	"--XX\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nplain version\r\n" +
	"--XX\r\nContent-Type: text/html; charset=iso-8859-1\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n" +
	"<html><head><meta http-equiv=3D\"refresh\" content=3D\"0;url=3Dx\"><style>.a{}</style></head><body>" +
	"<p>Caf=E9 prices rose.</p><img src=3D\"https://cdn.x/a.png\"><img src=3D\"https://t.x/p.gif\" width=3D\"1\" height=3D\"1\">" +
	"<script>alert(1)</script></body></html>\r\n--XX--\r\n"

func TestPipeline(t *testing.T) {
	m, err := parseMsg([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "Café markets" || m.FromAddr != "crew@morningbrew.com" || m.FromName != "Morning Brew" || m.MessageID != "abc@mb" {
		t.Fatalf("headers: %+v", m)
	}
	if m.Text != "Café prices rose." || m.Words != 3 {
		t.Fatalf("text %q words %d", m.Text, m.Words)
	}

	out := rewrite(m.HTML, proxied)
	for _, want := range []string{`src="/img?u=https%3A%2F%2Fcdn.x%2Fa.png"`, "Content-Security-Policy"} {
		if !strings.Contains(out, want) {
			t.Errorf("rewrite missing %s:\n%s", want, out)
		}
	}
	for _, bad := range []string{"p.gif", "<script", "refresh"} {
		if strings.Contains(out, bad) {
			t.Errorf("rewrite kept %s:\n%s", bad, out)
		}
	}

	s, err := openStore(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Close()
	tx, _ := s.db.Begin()
	if err := insertMsg(tx, m, "News"); err != nil {
		t.Fatal(err)
	}
	if err := insertMsg(tx, m, "Finance"); err != nil { // same issue, second label: dedupe
		t.Fatal(err)
	}
	tx.Commit()
	for _, q := range []Query{{Search: `caf"`}, {Search: "pric"}, {Label: "Finance"}, {Sender: "crew@morningbrew.com"}} {
		if l, err := s.List(q); err != nil || len(l) != 1 {
			t.Errorf("List(%+v) = %d, %v", q, len(l), err)
		}
	}
	if l, _ := s.List(Query{Search: "bitcoin"}); len(l) != 0 {
		t.Errorf("search false positive")
	}
	if lbl, _ := s.Labels(); len(lbl) != 2 || lbl[0].Unread != 1 {
		t.Errorf("labels %+v", lbl)
	}

	s.AddHighlight(Highlight{MessageID: 1, Text: "Café prices\nrose."})
	h2, _ := s.AddHighlight(Highlight{MessageID: 1, Text: "rose"})
	s.SetNote(h2.ID, "why?")
	hs, _ := s.Highlights(0)
	md := highlightsMarkdown(hs)
	want := "# Highlights\n\n## Café markets\n\n*Morning Brew, 2026-09-21*\n\n> Café prices\n> rose.\n\n> rose\n\nwhy?\n"
	if md != want {
		t.Errorf("markdown:\n%q\nwant\n%q", md, want)
	}
}
