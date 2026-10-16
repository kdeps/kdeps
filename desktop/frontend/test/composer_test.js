"use strict";
const assert = require("assert");
const c = require("../dist/composer.js");

assert.strictEqual(c.shouldComplete("/mod", 4), true);
assert.strictEqual(c.shouldComplete("hello", 5), false);
assert.strictEqual(c.shouldComplete("look at @src/ma", 15), true);
assert.strictEqual(c.shouldComplete("mail me@x.com", 13), false);

assert.strictEqual(c.caretOffset("a\u{1F600}b", 3), 2);

assert.deepStrictEqual(c.applyCompletion("/mo", 3, { replace: 3 }, "/model"), { text: "/model", pos: 6 });
assert.deepStrictEqual(c.applyCompletion("/model be", 9, { replace: 2 }, "beta"), { text: "/model beta", pos: 11 });
assert.deepStrictEqual(c.applyCompletion("/model be rest", 9, { replace: 2 }, "beta"), { text: "/model beta rest", pos: 11 });
assert.deepStrictEqual(c.applyCompletion("x @sr", 5, { replace: 3 }, "@src/"), { text: "x @src/", pos: 7 });

const models = [
  { name: "alpha", type: "gguf", repo: "org/alpha-GGUF", downloaded: true, enabled: true },
  { name: "beta", type: "llamafile", downloaded: false, enabled: true },
  { name: "llama3", type: "ollama", downloaded: true, enabled: true },
  { name: "gpt-4o", type: "cloud", downloaded: false, enabled: false },
  { name: "claude", type: "cloud", downloaded: false, enabled: true },
];
assert.strictEqual(c.filterModels(models, "all", "").length, 5);
assert.deepStrictEqual(c.filterModels(models, "cloud", "").map((m) => m.name), ["gpt-4o", "claude"]);
assert.deepStrictEqual(c.filterModels(models, "all", "ORG/").map((m) => m.name), ["alpha"]);
assert.deepStrictEqual(c.typeCounts(models), { all: 5, llamafile: 1, gguf: 1, ollama: 1, cloud: 2 });

assert.strictEqual(c.modelAction(models[0]), "use");
assert.strictEqual(c.modelAction(models[1]), "download");
assert.strictEqual(c.modelAction(models[2]), "use");
assert.strictEqual(c.modelAction(models[3]), "needs-key");
assert.strictEqual(c.modelAction(models[4]), "use");
console.log("composer ok");
