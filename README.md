# Newsletters

A small desktop reader for the newsletters your Gmail filters already sort into labels. It is not an email client: it only reads the labels you pick and never changes anything in your mailbox. Read state, "done" and highlights stay in the app.

## Setup

1. Turn on 2-step verification for your Google account, then create an app password at <https://myaccount.google.com/apppasswords>.
2. Make sure IMAP is enabled in Gmail (Settings → Forwarding and POP/IMAP).
3. Open the app, enter your Gmail address and the app password, click **Load labels from Gmail**, tick your newsletter labels, then **Save and sync**.

The first sync pulls the full history of those labels. After that it only fetches new mail, on launch and every 15 minutes (configurable).

## Using it

| Key | Action |
| --- | --- |
| `j` / `k` | Next / previous issue |
| `/` | Search (subject, sender, body) |
| `h` | Highlight selected text (or click the Highlight button) |
| `s` / `Shift+S` | Save issue as HTML / Markdown |
| `u` | Toggle read |
| `e` | Done (hides it; Undo in the toast) |
| `r` | Sync now |
| `g h` / `g i` / `g s` | Highlights / issues / settings |
| `Esc` | Close note, leave a text field |

Settings → Appearance switches between Comfortable and Compact density (narrower panes, tighter rows) and three text sizes; changes apply immediately.

Click a highlight to add a note or remove it. The Highlights page exports everything to `highlights.md` in your save folder, grouped by issue, or copies it to the clipboard.

Saved HTML files have their images embedded, so they open offline. Images in the reader are fetched by the app and cached on disk, and 1×1 tracking pixels are stripped. Scripts in emails never run.

## Where things live

`%APPDATA%\newsletter-reader\`: `config.json` (includes the app password in plain text), `cache.db` (SQLite), `img\` (image cache). Delete `cache.db` to resync from scratch. Set `NLR_DATA` to a folder to run with a separate profile.

## Development

Requires Go 1.26+, Node 20+, and the Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```sh
wails dev            # live reload
wails build          # build/bin/email_newsletter_reader.exe
go test ./...        # parsing, HTML rewrite, store, export, sync against an in-memory IMAP server
```

Known limits, marked `ponytail:` in the code: messages deleted or relabelled in Gmail stay in the local cache; the image cache never evicts; inline `cid:` images are not shown.
