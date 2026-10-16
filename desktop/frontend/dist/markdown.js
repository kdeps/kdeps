"use strict";
// Minimal offline Markdown: parseMarkdown() builds a plain AST, renderMarkdown()
// turns it into DOM with createElement/textContent only (never innerHTML), so
// model output cannot inject markup. Links are limited to http, https, mailto.

const SAFE_LINK = /^(https?:\/\/|mailto:)/i;

function parseInline(text) {
  const out = [];
  let buf = "";
  const flush = () => { if (buf) { out.push({ t: "text", v: buf }); buf = ""; } };
  let i = 0;
  while (i < text.length) {
    const rest = text.slice(i);
    let m;
    if ((m = /^`([^`\n]+)`/.exec(rest))) {
      flush(); out.push({ t: "code", v: m[1] }); i += m[0].length;
    } else if ((m = /^\*\*([^*\n]+)\*\*/.exec(rest)) || (m = /^__([^_\n]+)__/.exec(rest))) {
      flush(); out.push({ t: "strong", c: parseInline(m[1]) }); i += m[0].length;
    } else if ((m = /^\*([^*\s][^*\n]*)\*/.exec(rest))) {
      flush(); out.push({ t: "em", c: parseInline(m[1]) }); i += m[0].length;
    } else if ((m = /^\[([^\]\n]+)\]\(([^)\s]+)\)/.exec(rest))) {
      if (SAFE_LINK.test(m[2])) { flush(); out.push({ t: "link", href: m[2], c: parseInline(m[1]) }); }
      else { buf += m[1]; }
      i += m[0].length;
    } else {
      buf += text[i]; i++;
    }
  }
  flush();
  return out;
}

function parseMarkdown(src) {
  const lines = String(src).replace(/\r\n?/g, "\n").split("\n");
  const blocks = [];
  let para = [];
  const flushPara = () => {
    if (para.length) { blocks.push({ t: "p", c: parseInline(para.join("\n")) }); para = []; }
  };
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    let m;
    if ((m = /^```\s*([\w+-]*)\s*$/.exec(line))) {
      flushPara();
      const code = [];
      i++;
      while (i < lines.length && !/^```\s*$/.test(lines[i])) { code.push(lines[i]); i++; }
      blocks.push({ t: "pre", lang: m[1], v: code.join("\n") });
    } else if ((m = /^(#{1,6})\s+(.*)$/.exec(line))) {
      flushPara();
      blocks.push({ t: "h", n: m[1].length, c: parseInline(m[2]) });
    } else if (/^\s*([-*_])(\s*\1){2,}\s*$/.test(line)) {
      flushPara();
      blocks.push({ t: "hr" });
    } else if ((m = /^\s*([-*+]|\d+\.)\s+(.*)$/.exec(line))) {
      flushPara();
      const ordered = /\d/.test(m[1]);
      const items = [parseInline(m[2])];
      while (i + 1 < lines.length) {
        const nx = /^\s*([-*+]|\d+\.)\s+(.*)$/.exec(lines[i + 1]);
        if (!nx || /\d/.test(nx[1]) !== ordered) break;
        items.push(parseInline(nx[2]));
        i++;
      }
      blocks.push({ t: "list", ordered, items });
    } else if (/^>\s?/.test(line)) {
      flushPara();
      const q = [line.replace(/^>\s?/, "")];
      while (i + 1 < lines.length && /^>\s?/.test(lines[i + 1])) { i++; q.push(lines[i].replace(/^>\s?/, "")); }
      blocks.push({ t: "quote", c: parseInline(q.join("\n")) });
    } else if (line.trim() === "") {
      flushPara();
    } else {
      para.push(line);
    }
  }
  flushPara();
  return blocks;
}

function renderInline(parent, nodes) {
  for (const n of nodes) {
    if (n.t === "text") parent.appendChild(document.createTextNode(n.v));
    else if (n.t === "code") { const e = document.createElement("code"); e.textContent = n.v; parent.appendChild(e); }
    else if (n.t === "link") {
      const a = document.createElement("a");
      a.href = n.href; a.rel = "noopener noreferrer"; a.target = "_blank";
      renderInline(a, n.c); parent.appendChild(a);
    } else {
      const e = document.createElement(n.t);
      renderInline(e, n.c); parent.appendChild(e);
    }
  }
}

function renderMarkdown(target, src) {
  const frag = document.createDocumentFragment();
  for (const b of parseMarkdown(src)) {
    let e;
    if (b.t === "pre") {
      e = document.createElement("pre");
      const c = document.createElement("code");
      if (b.lang) c.dataset.lang = b.lang;
      c.textContent = b.v;
      e.appendChild(c);
    } else if (b.t === "h") {
      e = document.createElement("h" + Math.min(b.n + 2, 6));
      renderInline(e, b.c);
    } else if (b.t === "hr") {
      e = document.createElement("hr");
    } else if (b.t === "list") {
      e = document.createElement(b.ordered ? "ol" : "ul");
      for (const it of b.items) { const li = document.createElement("li"); renderInline(li, it); e.appendChild(li); }
    } else if (b.t === "quote") {
      e = document.createElement("blockquote");
      renderInline(e, b.c);
    } else {
      e = document.createElement("p");
      renderInline(e, b.c);
    }
    frag.appendChild(e);
  }
  target.replaceChildren(frag);
}

if (typeof module !== "undefined") module.exports = { parseMarkdown, parseInline };
