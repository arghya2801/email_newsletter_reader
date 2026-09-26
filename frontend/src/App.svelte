<script>
  import { onMount } from 'svelte'
  import { GetConfig, Labels, Senders, List, Open, SetRead, SetHidden, SaveEmail, Sync } from '../wailsjs/go/main/App'
  import { EventsOn } from '../wailsjs/runtime/runtime'
  import Reader from './Reader.svelte'
  import Highlights from './Highlights.svelte'
  import Settings from './Settings.svelte'

  let view = $state('reader') // reader | highlights | settings
  let q = $state({ label: '', sender: '', search: '', sort: 'new' })
  let items = $state([])
  let more = true
  let labels = $state([])
  let senders = $state([])
  let senderFilter = $state('')
  let issue = $state(null)
  let sync = $state({ running: false, count: 0, error: '' })
  let syncedAt = $state(null)
  let toast = $state(null)
  let reader = $state()
  let searchEl = $state()
  let listEl = $state()

  const sorts = [
    ['new', 'Newest first'], ['old', 'Oldest first'], ['sender', 'Sender A–Z'],
    ['unread', 'Unread first'], ['short', 'Shortest read'], ['long', 'Longest read'],
  ]

  const year = new Date().getFullYear()
  const fmtDate = (s) => {
    const d = new Date(s * 1000)
    return d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short', ...(d.getFullYear() !== year && { year: 'numeric' }) })
  }
  const mins = (w) => Math.max(1, Math.round(w / 230))
  const short = (l) => l.split('/').pop()

  let visibleSenders = $derived(
    senderFilter ? senders.filter((s) => (s.name + s.key).toLowerCase().includes(senderFilter.toLowerCase())) : senders,
  )

  async function load(reset) {
    const page = (await List({ ...q, offset: reset ? 0 : items.length })) ?? []
    items = reset ? page : [...items, ...page]
    more = page.length === 100
    if (reset && listEl) listEl.scrollTop = 0
  }

  async function counts() {
    labels = (await Labels()) ?? []
    senders = (await Senders(q.label)) ?? []
  }

  // Reload the list when any filter changes; search is debounced.
  let timer
  $effect(() => {
    const { label, sender, search, sort } = q
    clearTimeout(timer)
    timer = setTimeout(() => load(true), search ? 200 : 0)
  })
  $effect(() => {
    q.label
    Senders(q.label).then((s) => (senders = s ?? []))
  })

  function flash(text, action, fn) {
    toast = { text, action, fn }
    clearTimeout(flash.t)
    flash.t = setTimeout(() => (toast = null), 5000)
  }

  async function open(m) {
    issue = await Open(m.id)
    view = 'reader'
    if (!m.read) {
      m.read = true
      counts()
    }
  }

  function move(d) {
    const i = items.findIndex((m) => m.id === issue?.id)
    const next = items[i + d] ?? (i < 0 ? items[0] : null)
    if (next) {
      open(next)
      listEl?.querySelector(`[data-id="${next.id}"]`)?.scrollIntoView({ block: 'nearest' })
    }
    if (more && i + d >= items.length - 5) load(false)
  }

  async function save(fmt) {
    if (!issue) return
    try {
      flash('Saved to ' + (await SaveEmail(issue.id, fmt)))
    } catch (e) {
      flash('Could not save: ' + e)
    }
  }

  async function toggleRead() {
    if (!issue) return
    const m = items.find((x) => x.id === issue.id)
    const read = !(m?.read ?? issue.read)
    await SetRead(issue.id, read)
    if (m) m.read = read
    issue.read = read
    counts()
  }

  async function done() {
    if (!issue) return
    const id = issue.id
    const i = items.findIndex((m) => m.id === id)
    await SetHidden(id, true)
    const next = items[i + 1] ?? items[i - 1]
    items = items.filter((m) => m.id !== id)
    if (next) open(next)
    else issue = null
    counts()
    flash('Marked done.', 'Undo', async () => {
      await SetHidden(id, false)
      load(true)
      counts()
    })
  }

  let gPending = false
  function onkey(e) {
    if (e.target?.closest?.('input, textarea, select')) {
      if (e.key === 'Escape') e.target.blur()
      return
    }
    if (e.ctrlKey || e.metaKey || e.altKey) return
    if (gPending) {
      gPending = false
      if (e.key === 'h') view = 'highlights'
      if (e.key === 'i') view = 'reader'
      if (e.key === 's') view = 'settings'
      return
    }
    const keys = {
      j: () => move(1),
      k: () => move(-1),
      h: () => reader?.highlight(),
      s: () => save('html'),
      S: () => save('md'),
      e: done,
      u: toggleRead,
      r: Sync,
      '/': () => searchEl?.focus(),
      g: () => (gPending = true),
      Escape: () => reader?.escape(),
    }
    if (keys[e.key]) {
      e.preventDefault()
      keys[e.key]()
    }
  }

  onMount(async () => {
    const cfg = await GetConfig()
    if (!cfg.password || !cfg.labels?.length) view = 'settings'
    counts()
    return EventsOn('sync', (s) => {
      sync = s
      if (!s.running) syncedAt = new Date()
      if (!s.running || items.length < 100) {
        counts()
        load(true)
      }
    })
  })
</script>

<svelte:window onkeydown={onkey} />

<div class="app" class:wide={view !== 'reader'}>
  <nav class="side">
    <button class="row" class:on={view === 'reader' && !q.label} onclick={() => ((q.label = ''), (q.sender = ''), (view = 'reader'))}>
      <span>All newsletters</span>
    </button>
    {#each labels as l (l.key)}
      <button class="row" class:on={view === 'reader' && q.label === l.key} title={l.key}
        onclick={() => ((q.label = l.key), (q.sender = ''), (view = 'reader'))}>
        <span>{short(l.key)}</span>{#if l.unread}<span class="n">{l.unread}</span>{/if}
      </button>
    {/each}

    <div class="senders-head">
      <span class="muted">Senders</span>
      <input placeholder="Filter" aria-label="Filter senders" bind:value={senderFilter} />
    </div>
    <div class="senders">
      {#each visibleSenders as s (s.key)}
        <button class="row" class:on={q.sender === s.key} title={s.key}
          onclick={() => ((q.sender = q.sender === s.key ? '' : s.key), (view = 'reader'))}>
          <span>{s.name}</span>{#if s.unread}<span class="n">{s.unread}</span>{/if}
        </button>
      {/each}
    </div>

    <div class="foot">
      <button class="row" class:on={view === 'highlights'} onclick={() => (view = 'highlights')} title="g h">Highlights</button>
      <button class="row" class:on={view === 'settings'} onclick={() => (view = 'settings')} title="g s">Settings</button>
      <div class="sync">
        <span class="muted" role="status">
          {#if sync.running}Syncing{sync.label ? ` ${short(sync.label)}` : ''}… {sync.count ? `${sync.count} new` : ''}
          {:else if sync.error}Sync failed: {sync.error}
          {:else if syncedAt}Synced {syncedAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
          {/if}
        </span>
        <button onclick={Sync} disabled={sync.running} title="r">Sync now</button>
      </div>
    </div>
  </nav>

  {#if view === 'reader'}
    <section class="list">
      <div class="tools">
        <input type="search" placeholder="Search" aria-label="Search newsletters" bind:this={searchEl} bind:value={q.search} />
        <select bind:value={q.sort} aria-label="Sort">
          {#each sorts as [v, label]}<option value={v}>{label}</option>{/each}
        </select>
      </div>
      <div class="items" bind:this={listEl} onscroll={(e) => {
        const t = e.currentTarget
        if (more && t.scrollTop + t.clientHeight > t.scrollHeight - 400) { more = false; load(false) }
      }}>
        {#each items as m (m.id)}
          <button class="item" class:unread={!m.read} class:sel={issue?.id === m.id} data-id={m.id} onclick={() => open(m)}>
            <span class="meta"><span class="from">{m.fromName}</span><span>{fmtDate(m.date)}</span></span>
            <span class="subj">{m.subject || '(no subject)'}</span>
            <span class="meta">{mins(m.words)} min read</span>
          </button>
        {:else}
          <p class="empty muted">
            {#if q.search || q.sender}Nothing matches. Clear the search or pick another sender.
            {:else if sync.running}Fetching your newsletters…
            {:else}No newsletters yet. Pick your labels in Settings, then sync.{/if}
          </p>
        {/each}
      </div>
    </section>
  {/if}

  <main class="main">
    {#if view === 'settings'}
      <Settings onsaved={() => { view = 'reader'; Sync() }} />
    {:else if view === 'highlights'}
      <Highlights onopen={(id) => open({ id, read: true })} />
    {:else if issue}
      {#key issue.id}
        <Reader bind:this={reader} {issue} {onkey} onsave={save} ontoggleread={toggleRead} ondone={done} />
      {/key}
    {:else}
      <p class="empty muted">Pick an issue on the left, or press j.</p>
    {/if}
  </main>

  {#if toast}
    <div class="toast" role="status">
      <span>{toast.text}</span>
      {#if toast.action}<button onclick={() => { toast.fn(); toast = null }}>{toast.action}</button>{/if}
    </div>
  {/if}
</div>

<style>
  .app {
    display: grid;
    grid-template-columns: 232px 360px 1fr;
    height: 100vh;
  }
  .app.wide { grid-template-columns: 232px 1fr; }

  .side {
    display: flex; flex-direction: column; min-height: 0;
    padding: 16px 8px 8px; border-right: 1px solid var(--rule);
  }
  .row {
    display: flex; justify-content: space-between; gap: 8px; width: 100%;
    text-align: left; padding: 5px 10px;
  }
  .row span:first-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row.on { background: var(--paper); }
  .n { color: var(--muted); font-variant-numeric: tabular-nums; }
  .senders-head { display: flex; align-items: center; gap: 8px; padding: 20px 10px 6px; }
  .senders-head input { flex: 1; min-width: 0; padding: 2px 6px; }
  .senders { flex: 1; overflow-y: auto; min-height: 0; }
  .foot { border-top: 1px solid var(--rule); padding-top: 8px; }
  .sync { display: flex; align-items: center; justify-content: space-between; gap: 6px; padding: 6px 10px 0; font-size: 12px; }
  .sync span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sync button { flex: none; font-size: 12px; }

  .list { display: flex; flex-direction: column; min-height: 0; border-right: 1px solid var(--rule); }
  .tools { display: flex; gap: 8px; padding: 12px; border-bottom: 1px solid var(--rule); }
  .tools input { flex: 1; min-width: 0; }
  .items { overflow-y: auto; flex: 1; }
  .item {
    display: flex; flex-direction: column; gap: 3px; width: 100%; text-align: left;
    padding: 12px 16px; border-radius: 0; border-bottom: 1px solid var(--rule);
  }
  .item:hover { background: color-mix(in srgb, var(--paper) 50%, var(--chrome)); }
  .item.sel { background: var(--paper); }
  .meta { display: flex; justify-content: space-between; gap: 8px; font-size: 12px; color: var(--muted); }
  .from { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .subj {
    font: 400 17px/1.3 var(--serif); font-optical-sizing: auto;
    color: color-mix(in srgb, var(--ink) 75%, var(--chrome));
  }
  .unread .subj { font-weight: 600; color: var(--ink); }
  .unread .from { color: var(--ink); }

  .main { overflow-y: auto; min-height: 0; }
  .empty { padding: 32px 24px; }

  .toast {
    position: fixed; bottom: 16px; left: 50%; transform: translateX(-50%);
    display: flex; gap: 12px; align-items: center; max-width: 70vw;
    background: var(--ink); color: var(--paper); padding: 8px 14px; border-radius: 6px;
  }
  .toast span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .toast button { color: var(--paper); text-decoration: underline; }
  .toast button:hover { background: transparent; }
</style>
