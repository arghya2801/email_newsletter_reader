package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
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

func TestFetchImageCaches(t *testing.T) {
	png := "\x89PNG\r\n\x1a\nfake"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(png)) }))
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "img"), 0o755)
	for i := 0; i < 2; i++ { // second call must come from disk
		b, ct, err := fetchImage(dir, srv.URL+"/a.png")
		if err != nil || string(b) != png || ct != "image/png" {
			t.Fatalf("call %d: %q %q %v", i, b, ct, err)
		}
		srv.Close()
	}
	if _, _, err := fetchImage(dir, "file:///c:/windows/win.ini"); err == nil {
		t.Error("non-http url accepted")
	}
}

// TestSync runs the real sync against go-imap's in-memory server: batching,
// resume from last UID, dedupe across labels, and that the mailbox is untouched.
func TestSync(t *testing.T) {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("me", "pw")
	mem.AddUser(u)
	u.Create("News", nil)
	u.Create("Fin", nil)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: true,
	})
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	defer srv.Close()

	cl, err := imapclient.DialInsecure(ln.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	cl.Login("me", "pw").Wait()
	add := func(box string, i int) {
		raw := []byte(strings.Replace(fixture, "<abc@mb>", fmt.Sprintf("<m%d@x>", i), 1))
		cmd := cl.Append(box, int64(len(raw)), nil)
		cmd.Write(raw)
		cmd.Close()
		if _, err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 120; i++ {
		add("News", i)
	}
	add("Fin", 7) // same issue filed under two labels

	s, _ := openStore(filepath.Join(t.TempDir(), "t.db"))
	defer s.db.Close()
	count := func() (n int) { s.db.QueryRow(`SELECT count(*) FROM messages`).Scan(&n); return }
	var calls []int
	if err := s.syncLabel(cl, "News", func(n int) { calls = append(calls, n) }); err != nil {
		t.Fatal(err)
	}
	if count() != 120 || fmt.Sprint(calls) != "[50 100 120]" {
		t.Fatalf("first sync: %d rows, progress %v", count(), calls)
	}
	add("News", 500)
	calls = nil
	s.syncLabel(cl, "News", func(n int) { calls = append(calls, n) })
	if count() != 121 || fmt.Sprint(calls) != "[1]" {
		t.Fatalf("resume: %d rows, progress %v", count(), calls)
	}
	s.syncLabel(cl, "Fin", func(int) {})
	if l, _ := s.List(Query{Label: "Fin"}); count() != 121 || len(l) != 1 {
		t.Fatalf("dedupe: %d rows, Fin has %d", count(), len(l))
	}

	cl.Select("News", nil).Wait()
	msgs, _ := cl.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{Flags: true}).Collect()
	if len(msgs[0].Flags) != 0 {
		t.Errorf("sync changed server flags: %v", msgs[0].Flags)
	}
}

// TestMigrateV1 opens a cache written by the first schema and checks nothing is lost.
func TestMigrateV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, _ := sql.Open("sqlite", path)
	for _, q := range []string{
		`CREATE TABLE messages(id INTEGER PRIMARY KEY, message_id TEXT UNIQUE NOT NULL, from_name TEXT NOT NULL, from_addr TEXT NOT NULL, subject TEXT NOT NULL, date INTEGER NOT NULL, html TEXT NOT NULL, text TEXT NOT NULL, words INTEGER NOT NULL, read INTEGER NOT NULL DEFAULT 0, hidden INTEGER NOT NULL DEFAULT 0)`,
		`CREATE INDEX messages_date ON messages(date)`,
		`CREATE TABLE message_labels(message_id INTEGER NOT NULL REFERENCES messages(id), label TEXT NOT NULL, PRIMARY KEY(message_id, label))`,
		`CREATE TABLE highlights(id INTEGER PRIMARY KEY, message_id INTEGER NOT NULL REFERENCES messages(id), text TEXT NOT NULL, prefix TEXT NOT NULL, suffix TEXT NOT NULL, note TEXT NOT NULL DEFAULT '', created INTEGER NOT NULL)`,
		`CREATE VIRTUAL TABLE fts USING fts5(subject, from_name, text, content='messages', content_rowid='id')`,
		`INSERT INTO messages VALUES(7,'<a@b>','Brew','b@x','Coffee prices',100,'<p>Arabica beans surged</p>','Arabica beans surged',3,1,0)`,
		`INSERT INTO fts(rowid,subject,from_name,text) VALUES(7,'Coffee prices','Brew','Arabica beans surged')`,
		`INSERT INTO message_labels VALUES(7,'News')`,
		`INSERT INTO highlights VALUES(1,7,'beans','Arabica ','','keep',5)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	old.Close()

	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Close()
	m, err := s.Get(7)
	if err != nil || m.HTML != "<p>Arabica beans surged</p>" || !m.Read {
		t.Fatalf("Get after migrate: %+v %v", m, err)
	}
	if l, _ := s.List(Query{Label: "News", Search: "arabica"}); len(l) != 1 {
		t.Errorf("label+search after migrate: %d", len(l))
	}
	if hs, _ := s.Highlights(7); len(hs) != 1 || hs[0].Note != "keep" {
		t.Errorf("highlights after migrate: %+v", hs)
	}
	var fk int
	s.db.QueryRow(`SELECT count(*) FROM pragma_foreign_key_check`).Scan(&fk)
	if fk != 0 {
		t.Errorf("%d foreign key violations", fk)
	}
	s.db.Close()
	if s, err = openStore(path); err != nil { // second open is a no-op
		t.Fatal(err)
	}
	s.db.Close()
}
