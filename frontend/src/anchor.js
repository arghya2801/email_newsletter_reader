// Highlight anchoring: a highlight is stored as its exact text plus up to 32
// chars of text before/after, and re-found by searching the document's text.
// ponytail: if the newsletter HTML changes, or the quote repeats with the same
// prefix, the first match wins; fine for immutable emails.

const textNodes = (root) => {
  const w = root.ownerDocument.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  const out = []
  for (let n; (n = w.nextNode()); ) out.push(n)
  return out
}

export function context(doc, r) {
  const pre = doc.createRange()
  pre.setStart(doc.body, 0)
  pre.setEnd(r.startContainer, r.startOffset)
  const post = doc.createRange()
  post.setStart(r.endContainer, r.endOffset)
  post.setEnd(doc.body, doc.body.childNodes.length)
  return { prefix: pre.toString().slice(-32), suffix: post.toString().slice(0, 32) }
}

export function findRange(doc, h) {
  const nodes = textNodes(doc.body)
  let full = ''
  const starts = nodes.map((n) => ((full += n.data), full.length - n.data.length))
  let at = -1
  for (let i = full.indexOf(h.text); i >= 0; i = full.indexOf(h.text, i + 1)) {
    if (at < 0) at = i
    if (full.slice(Math.max(0, i - h.prefix.length), i) === h.prefix) {
      at = i
      break
    }
  }
  if (at < 0) return null
  const r = doc.createRange()
  const end = at + h.text.length
  const si = starts.findIndex((s, i) => at < s + nodes[i].length)
  const ei = starts.findIndex((s, i) => end <= s + nodes[i].length)
  r.setStart(nodes[si], at - starts[si])
  r.setEnd(nodes[ei], end - starts[ei])
  return r
}

export function paint(doc, r, id, fresh) {
  let root = r.commonAncestorContainer
  if (root.nodeType === Node.TEXT_NODE) root = root.parentNode
  const hit = textNodes(root).filter((n) => r.intersectsNode(n))
  const [sc, so, ec, eo] = [r.startContainer, r.startOffset, r.endContainer, r.endOffset]
  for (let n of hit) {
    const s = n === sc ? so : 0
    const e = n === ec ? eo : n.length
    if (e <= s || !n.data.slice(s, e).trim() || n.parentNode.closest('style, script, title')) continue
    if (e < n.length) n.splitText(e)
    if (s > 0) n = n.splitText(s)
    const m = doc.createElement('mark')
    m.className = 'nl-hl' + (fresh ? ' fresh' : '')
    m.dataset.id = id
    n.parentNode.insertBefore(m, n)
    m.appendChild(n)
  }
}

export function unpaint(doc, id) {
  for (const m of doc.querySelectorAll(`mark.nl-hl[data-id="${id}"]`)) {
    const p = m.parentNode
    m.replaceWith(...m.childNodes)
    p.normalize()
  }
}
