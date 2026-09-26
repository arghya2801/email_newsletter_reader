<script>
  import { onMount } from 'svelte'
  import { Highlights, SetNote, DeleteHighlight, ExportHighlights } from '../wailsjs/go/main/App'
  import { ClipboardSetText } from '../wailsjs/runtime/runtime'

  let { onopen } = $props()
  let all = $state([])
  let status = $state('')

  // Highlights arrive ordered by issue, so grouping is one pass.
  let groups = $derived(
    all.reduce((g, h) => {
      if (g.at(-1)?.id !== h.messageId) g.push({ id: h.messageId, subject: h.subject, from: h.fromName, date: h.date, items: [] })
      g.at(-1).items.push(h)
      return g
    }, []),
  )

  onMount(async () => (all = (await Highlights()) ?? []))

  const fmt = (s) => new Date(s * 1000).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })

  async function exportMd(copy) {
    try {
      const ex = await ExportHighlights()
      if (copy) {
        await ClipboardSetText(ex.markdown)
        status = 'Copied to clipboard. Also saved to ' + ex.path
      } else status = 'Saved to ' + ex.path
    } catch (e) {
      status = 'Could not export: ' + e
    }
  }

  async function saveNote(h, note) {
    if (note === h.note) return
    await SetNote(h.id, note)
    h.note = note
  }

  async function remove(h) {
    await DeleteHighlight(h.id)
    all = all.filter((x) => x.id !== h.id)
  }
</script>

<div class="book">
  <header>
    <h1>Highlights</h1>
    <div class="actions">
      <button onclick={() => exportMd(false)} disabled={!all.length}>Export to Markdown</button>
      <button onclick={() => exportMd(true)} disabled={!all.length}>Copy as Markdown</button>
    </div>
    {#if status}<p class="muted status" role="status">{status}</p>{/if}
  </header>

  {#each groups as g (g.id)}
    <section>
      <button class="issue" onclick={() => onopen(g.id)}>
        <span class="subj">{g.subject || '(no subject)'}</span>
        <span class="muted">{g.from}, {fmt(g.date)}</span>
      </button>
      {#each g.items as h (h.id)}
        <div class="quote">
          <blockquote><span class="marker">{h.text.trim()}</span></blockquote>
          <textarea rows="1" placeholder="Add a note" aria-label="Note" value={h.note}
            onblur={(e) => saveNote(h, e.currentTarget.value)}></textarea>
          <button class="rm muted" onclick={() => remove(h)}>Remove</button>
        </div>
      {/each}
    </section>
  {:else}
    <p class="muted">No highlights yet. Select text in any issue and press h, or click Highlight.</p>
  {/each}
</div>

<style>
  .book { max-width: 680px; margin: 0 auto; padding: 40px 32px 80px; }
  header { margin-bottom: 40px; }
  h1 { font: 500 36px/1.1 var(--serif); font-optical-sizing: auto; margin: 0 0 12px; }
  .actions { display: flex; gap: 4px; margin-left: -8px; }
  .status { font-size: 13px; overflow-wrap: anywhere; }
  section { margin-bottom: 44px; }
  .issue {
    display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
    padding: 0 0 12px; margin-bottom: 12px; width: 100%; text-align: left;
    border-bottom: 1px solid var(--rule); border-radius: 0;
  }
  .issue:hover { background: none; }
  .issue:hover .subj { text-decoration: underline; text-decoration-thickness: 1px; text-underline-offset: 3px; }
  .subj { font: 500 21px/1.25 var(--serif); font-optical-sizing: auto; }
  .issue .muted { font-size: 13px; }
  .quote { position: relative; margin: 18px 0; }
  blockquote { margin: 0; font: 400 19px/1.6 var(--serif); font-optical-sizing: auto; white-space: pre-line; }
  textarea {
    display: block; width: 100%; margin-top: 6px; field-sizing: content; resize: none;
    background: transparent; border-color: transparent; padding: 4px 0; color: var(--muted);
  }
  textarea:hover, textarea:focus { border-color: var(--rule); padding: 4px 8px; }
  .rm { position: absolute; right: -8px; bottom: -26px; font-size: 12px; opacity: 0; }
  .quote:hover .rm, .rm:focus-visible { opacity: 1; }
</style>
