<script>
  import { AddHighlight, SetNote, DeleteHighlight } from '../wailsjs/go/main/App'
  import { BrowserOpenURL } from '../wailsjs/runtime/runtime'
  import { findRange, paint, unpaint, context } from './anchor.js'

  let { issue, onkey, zoom, onzoom, onsave, ontoggleread, ondone } = $props()

  let frame = $state()
  let highlights = $state([...(issue.highlights ?? [])])
  let pop = $state(null) // { kind: 'new' | 'note', x, y, h? }

  // Newsletters are designed for white; keep that, and style marks inside the frame.
  const inject = `<style>
    html{background:#fff;overflow:hidden}
    mark.nl-hl{color:inherit;cursor:pointer;border-radius:.8em .3em;padding:.05em .1em;
      -webkit-box-decoration-break:clone;box-decoration-break:clone;
      background:linear-gradient(104deg,transparent .9%,rgba(242,201,76,.55) 2.4%,rgba(242,201,76,.2) 5.8%,rgba(242,201,76,.55) 93%,rgba(242,201,76,.55) 96%,transparent 98%),
        linear-gradient(183deg,transparent 0%,rgba(242,201,76,.2) 8%,transparent 15%)}
    mark.nl-hl.fresh{animation:ink .45s ease-out}
    @keyframes ink{from{background-size:0 100%,0 100%}to{background-size:100% 100%,100% 100%}}
    @media (prefers-reduced-motion:reduce){mark.nl-hl.fresh{animation:none}}
  </style>`
  const srcdoc = issue.html.includes('</head>') ? issue.html.replace('</head>', inject + '</head>') : inject + issue.html

  const date = new Date(issue.date * 1000).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })

  // Size the frame to its content so the pane scrolls, not the frame. Zoom is applied
  // inside the frame; fixed-width newsletters zoomed past the pane widen it instead of clipping.
  // Measures from zero (scrollHeight never reports less than the frame) and keeps the reading spot.
  function fit() {
    const el = frame?.contentDocument?.documentElement
    if (!el) return
    const pane = frame.closest('.main')
    const top = pane ? frame.getBoundingClientRect().top - pane.getBoundingClientRect().top + pane.scrollTop : 0
    const read = pane ? pane.scrollTop - top : 0
    const before = frame.offsetHeight
    el.style.zoom = zoom / 100
    frame.style.minWidth = frame.style.height = '0'
    frame.style.minWidth = el.scrollWidth > el.clientWidth ? el.scrollWidth + 'px' : ''
    frame.style.height = el.scrollHeight + 'px'
    if (pane && read > 0 && before) pane.scrollTop = top + (read * frame.offsetHeight) / before
  }
  $effect(() => {
    zoom
    fit()
  })

  // Ctrl+wheel zooms the newsletter; pixel deltas (touchpads) add up to one step per notch.
  let wheelSum = 0
  function onwheel(e) {
    if (!e.ctrlKey) return
    e.preventDefault()
    wheelSum += e.deltaY
    if (Math.abs(wheelSum) >= 50) {
      onzoom(wheelSum < 0 ? 1 : -1)
      wheelSum = 0
    }
  }

  function loaded() {
    const doc = frame.contentDocument
    fit()
    new ResizeObserver(fit).observe(doc.body)
    doc.addEventListener('wheel', onwheel, { passive: false })
    doc.addEventListener('click', (e) => {
      const mark = e.target.closest?.('mark.nl-hl')
      if (mark) return openNote(mark)
      if (pop?.kind === 'note') closeNote()
      const a = e.target.closest?.('a[href]')
      if (a) {
        e.preventDefault()
        if (/^(https?|mailto):/i.test(a.href)) BrowserOpenURL(a.href)
      }
    })
    doc.addEventListener('mouseup', () => setTimeout(offer))
    doc.addEventListener('keydown', onkey)
    for (const h of highlights) {
      const r = findRange(doc, h)
      if (r) paint(doc, r, h.id)
    }
  }

  function selection() {
    const sel = frame?.contentDocument?.getSelection()
    if (!sel?.rangeCount || sel.isCollapsed) return null
    const r = sel.getRangeAt(0)
    return r.toString().trim().length > 1 ? r : null
  }

  function offer() {
    const r = selection()
    if (!r) {
      if (pop?.kind === 'new') pop = null
      return
    }
    const b = r.getBoundingClientRect()
    pop = { kind: 'new', x: b.left + b.width / 2, y: b.top }
  }

  function openNote(mark) {
    const h = highlights.find((x) => x.id === +mark.dataset.id)
    if (!h) return
    const b = mark.getBoundingClientRect()
    pop = { kind: 'note', x: b.left + b.width / 2, y: b.bottom, h, note: h.note }
  }

  export async function highlight() {
    const r = selection()
    if (!r) return
    const doc = frame.contentDocument
    const h = await AddHighlight({ messageId: issue.id, text: r.toString(), ...context(doc, r), note: '' })
    paint(doc, r, h.id, true)
    doc.getSelection().removeAllRanges()
    highlights.push(h)
    openNote(doc.querySelector(`mark[data-id="${h.id}"]`))
  }

  async function closeNote() {
    if (pop?.kind === 'note' && pop.note !== pop.h.note) {
      await SetNote(pop.h.id, pop.note)
      pop.h.note = pop.note
    }
    pop = null
  }

  async function remove() {
    const id = pop.h.id
    await DeleteHighlight(id)
    unpaint(frame.contentDocument, id)
    highlights = highlights.filter((x) => x.id !== id)
    pop = null
  }

  // Called by the app on Escape; true if something was closed.
  export function escape() {
    if (!pop) return false
    closeNote()
    return true
  }
</script>

<article class="sheet">
  <header>
    <h1>{issue.subject || '(no subject)'}</h1>
    <p class="by muted">
      <span title={issue.fromAddr}>{issue.fromName}</span>, {date}. {Math.max(1, Math.round(issue.words / 230))} min read
    </p>
    <div class="actions">
      <button onclick={() => onsave('html')} title="s">Save as HTML</button>
      <button onclick={() => onsave('md')} title="Shift+S">Save as Markdown</button>
      <button onclick={ontoggleread} title="u">Mark unread</button>
      <button onclick={ondone} title="e">Done</button>
      <span class="right">
        {#if zoom !== 100}<button class="muted zoom" onclick={() => onzoom(0)} title="Reset zoom (Ctrl+0)">{zoom}%</button>{/if}
        {#if highlights.length}<span class="muted count">{highlights.length} highlight{highlights.length > 1 ? 's' : ''}</span>{/if}
      </span>
    </div>
  </header>

  <div class="body" {onwheel}>
    <iframe bind:this={frame} title={issue.subject} sandbox="allow-same-origin" {srcdoc} onload={loaded}></iframe>

    {#if pop?.kind === 'new'}
      <button class="pop hl" style:left="{pop.x}px" style:top="{pop.y}px" onmousedown={(e) => e.preventDefault()} onclick={highlight} title="h">
        Highlight
      </button>
    {:else if pop?.kind === 'note'}
      <div class="pop note" style:left="{pop.x}px" style:top="{pop.y}px" role="dialog" aria-label="Highlight note">
        <!-- svelte-ignore a11y_autofocus -->
        <textarea bind:value={pop.note} placeholder="Add a note (optional)" rows="3" autofocus
          onkeydown={(e) => (e.key === 'Escape' || (e.key === 'Enter' && e.ctrlKey)) && (e.stopPropagation(), closeNote())}></textarea>
        <div class="row">
          <button onclick={remove}>Remove highlight</button>
          <button onclick={closeNote}>Done</button>
        </div>
      </div>
    {/if}
  </div>
</article>

<style>
  .sheet {
    max-width: 820px; margin: var(--sheet-m) auto 64px; width: calc(100% - 2 * var(--sheet-m)); background: var(--paper);
    border: 1px solid var(--rule); border-radius: 3px;
  }
  header { padding: var(--head-p); }
  h1 {
    font: 500 var(--h1-size)/1.15 var(--serif); font-optical-sizing: auto; letter-spacing: -0.01em;
    margin: 0 0 10px; text-wrap: balance;
  }
  .by { margin: 0 0 16px; }
  .actions { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; margin-left: -8px; }
  .right { margin-left: auto; display: flex; align-items: center; gap: 8px; font-size: 12px; }
  .zoom { font-size: 12px; font-variant-numeric: tabular-nums; }
  .body { position: relative; }
  iframe { display: block; width: 100%; height: 150px; border: 0; border-top: 1px solid var(--rule); background: #fff; }
  .pop {
    position: absolute; transform: translate(-50%, calc(-100% - 8px)); z-index: 2;
    background: var(--ink); color: var(--paper); border-radius: 5px;
  }
  .pop.hl:hover { background: var(--ink); text-decoration: underline; }
  .pop.note {
    transform: translate(-50%, 8px); width: 320px; padding: 8px; display: flex; flex-direction: column; gap: 6px;
    background: var(--paper); color: var(--ink); border: 1px solid var(--rule);
  }
  .pop.note textarea { resize: vertical; }
  .pop .row { display: flex; justify-content: space-between; }
</style>
