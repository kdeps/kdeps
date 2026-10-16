"use strict";
const assert = require("assert");
const { parseMarkdown, parseInline } = require("../dist/markdown.js");

const kinds = (src) => parseMarkdown(src).map((b) => b.t);

assert.deepStrictEqual(kinds("# T\n\ntext\n\n- a\n- b\n\n1. x\n2. y\n\n> q\n\n---"), ["h", "p", "list", "list", "quote", "hr"]);

const fence = parseMarkdown("```go\nfmt.Println(1)\n```\nafter");
assert.strictEqual(fence[0].t, "pre");
assert.strictEqual(fence[0].lang, "go");
assert.strictEqual(fence[0].v, "fmt.Println(1)");
assert.strictEqual(fence[1].t, "p");

// An unterminated fence (mid-stream) renders as code to the end.
assert.strictEqual(parseMarkdown("```\npartial")[0].v, "partial");

// Markup inside code is never interpreted.
const inl = parseInline("use `**x**` and **b** and *i*");
assert.deepStrictEqual(inl.map((n) => n.t), ["text", "code", "text", "strong", "text", "em"]);
assert.strictEqual(inl[1].v, "**x**");

// Unsafe link schemes are dropped to plain text.
assert.deepStrictEqual(parseInline("[x](javascript:alert(1))").map((n) => n.t), ["text"]);
assert.strictEqual(parseInline("[x](https://a.b)")[0].t, "link");

// Raw HTML stays text.
assert.deepStrictEqual(parseInline("<img src=x onerror=alert(1)>"), [{ t: "text", v: "<img src=x onerror=alert(1)>" }]);

// Ordered and unordered lists do not merge.
assert.strictEqual(parseMarkdown("- a\n1. b").length, 2);

console.log("markdown ok");
