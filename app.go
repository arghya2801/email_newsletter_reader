package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the whole frontend API; every exported method is bound to JS.
type App struct {
	ctx     context.Context
	dir     string
	store   *Store
	mu      sync.Mutex // guards cfg
	cfg     Config
	syncing sync.Mutex
}

func NewApp(dir string, store *Store) *App {
	return &App{dir: dir, store: store, cfg: loadConfig(dir)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		for {
			a.Sync()
			m := max(a.config().SyncMinutes, 1)
			time.Sleep(time.Duration(m) * time.Minute)
		}
	}()
}

func (a *App) config() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

func (a *App) GetConfig() Config { return a.config() }

func (a *App) SaveConfig(c Config) error {
	if c.Host == "" {
		c.Host = "imap.gmail.com:993"
	}
	a.mu.Lock()
	a.cfg = c
	a.mu.Unlock()
	return saveConfig(a.dir, c)
}

func (a *App) ServerLabels() ([]string, error) { return serverLabels(a.config()) }

func (a *App) ChooseSaveDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Folder for saved newsletters", DefaultDirectory: a.config().SaveDir})
}

type SyncStatus struct {
	Running bool   `json:"running"`
	Label   string `json:"label"`
	Count   int    `json:"count"`
	Error   string `json:"error"`
}

// Sync pulls new mail from every picked label. Concurrent calls are no-ops.
func (a *App) Sync() {
	if !a.syncing.TryLock() {
		return
	}
	defer a.syncing.Unlock()
	c := a.config()
	if len(c.Labels) == 0 || c.Password == "" {
		return
	}
	emit := func(s SyncStatus) { runtime.EventsEmit(a.ctx, "sync", s) }
	emit(SyncStatus{Running: true})
	cl, err := dial(c)
	if err != nil {
		emit(SyncStatus{Error: err.Error()})
		return
	}
	defer cl.Logout()
	fetched := 0 // new messages across all labels
	for _, l := range c.Labels {
		done := 0
		err := a.store.syncLabel(cl, l, func(n int) {
			done = n
			emit(SyncStatus{Running: true, Label: l, Count: fetched + n})
		})
		fetched += done
		if err != nil {
			emit(SyncStatus{Error: err.Error(), Count: fetched})
			return
		}
	}
	emit(SyncStatus{Count: fetched})
}

func (a *App) Labels() ([]Count, error)              { return a.store.Labels() }
func (a *App) Senders(label string) ([]Count, error) { return a.store.Senders(label) }
func (a *App) List(q Query) ([]Msg, error)           { return a.store.List(q) }
func (a *App) SetRead(id int64, read bool) error     { return a.store.SetRead(id, read) }
func (a *App) SetHidden(id int64, hidden bool) error { return a.store.SetHidden(id, hidden) }

type Issue struct {
	Msg
	Highlights []Highlight `json:"highlights"`
}

// Open returns an issue ready to render and marks it read (locally only).
func (a *App) Open(id int64) (Issue, error) {
	m, err := a.store.Get(id)
	if err != nil {
		return Issue{}, err
	}
	m.HTML = rewrite(m.HTML, proxied)
	a.store.SetRead(id, true)
	hs, err := a.store.Highlights(id)
	return Issue{m, hs}, err
}

func (a *App) SaveEmail(id int64, format string) (string, error) {
	m, err := a.store.Get(id)
	if err != nil {
		return "", err
	}
	return saveEmail(a.dir, a.config().SaveDir, m, format)
}

func (a *App) AddHighlight(h Highlight) (Highlight, error) { return a.store.AddHighlight(h) }
func (a *App) SetNote(id int64, note string) error         { return a.store.SetNote(id, note) }
func (a *App) DeleteHighlight(id int64) error              { return a.store.DeleteHighlight(id) }
func (a *App) Highlights() ([]Highlight, error)            { return a.store.Highlights(0) }

type Export struct {
	Path     string `json:"path"`
	Markdown string `json:"markdown"`
}

// ExportHighlights writes highlights.md to the save folder and returns it for the clipboard.
func (a *App) ExportHighlights() (Export, error) {
	hs, err := a.store.Highlights(0)
	if err != nil {
		return Export{}, err
	}
	md := highlightsMarkdown(hs)
	dir := a.config().SaveDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Export{}, err
	}
	p := filepath.Join(dir, "highlights.md")
	return Export{p, md}, os.WriteFile(p, []byte(md), 0o644)
}
