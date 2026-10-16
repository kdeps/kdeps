"use strict";
const assert = require("assert");
const p = require("../dist/projectsctl.js");

const g = p.groupProjects([
  { name: "zeta", installed: false }, { name: "alpha", installed: false }, { name: "pkg", installed: true },
]);
assert.deepStrictEqual(g.workspace.map((x) => x.name), ["alpha", "zeta"]);
assert.deepStrictEqual(g.installed.map((x) => x.name), ["pkg"]);
assert.deepStrictEqual(p.groupProjects(null), { workspace: [], installed: [] });

const inputs = [
  { name: "who", type: "string", required: true },
  { name: "n", type: "integer" },
  { name: "x", type: "number" },
  { name: "loud", type: "boolean" },
];
assert.deepStrictEqual(p.componentBody(inputs, { who: "ada", n: "3", x: "1.5", loud: true }), { who: "ada", n: 3, x: 1.5, loud: true });
assert.deepStrictEqual(p.componentBody(inputs, { who: "ada", n: "" }), { who: "ada" });
assert.throws(() => p.componentBody(inputs, {}), /who is required/);
assert.throws(() => p.componentBody(inputs, { who: "a", n: "1.5" }), /whole number/);
assert.throws(() => p.componentBody(inputs, { who: "a", x: "abc" }), /must be a number/);

const kind = { key: "chat", fields: [{ name: "model", type: "string" }, { name: "prompt", type: "text" }, { name: "tools", type: "yaml" }] };
assert.deepStrictEqual(p.shownFields(kind, { prompt: "hi" }, ["model"]).map((f) => f.name), ["model", "prompt"]);
assert.deepStrictEqual(p.addableFields(kind, { prompt: "hi" }).map((f) => f.name), ["model", "tools"]);
assert.deepStrictEqual(p.shownFields(null, {}), []);

assert.strictEqual(p.controlFor({ type: "boolean" }), "checkbox");
assert.strictEqual(p.controlFor({ type: "enum" }), "select");
assert.strictEqual(p.controlFor({ type: "yaml" }), "textarea");
assert.strictEqual(p.controlFor({ type: "text" }), "textarea");
assert.strictEqual(p.controlFor({ type: "integer" }), "number");
assert.strictEqual(p.controlFor({ type: "string" }), "text");

assert.strictEqual(p.formFieldValue({ type: "boolean" }, true), true);
assert.strictEqual(p.formFieldValue({ type: "boolean" }, "on"), false);
assert.strictEqual(p.formFieldValue({ type: "integer" }, 5), "5");
assert.strictEqual(p.formFieldValue({ type: "string" }, null), "");

assert.strictEqual(p.curlFor("http://127.0.0.1:3000/", { path: "/api/v1/x", methods: ["GET"] }),
  'curl -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" http://127.0.0.1:3000/api/v1/x');
assert.strictEqual(p.curlFor("http://h", { path: "/p", methods: ["POST"] }),
  "curl -X POST -H \"Authorization: Bearer $KDEPS_API_AUTH_TOKEN\" -H 'Content-Type: application/json' -d '{}' http://h/p");
assert.ok(p.curlFor("http://h", { path: "/p" }).startsWith("curl -H"), "no methods means GET");

assert.strictEqual(p.progressLabel({ status: "resource.started", summary: "ask" }), "ask: running");
assert.strictEqual(p.progressLabel({ status: "resource.failed", summary: "ask", text: "timeout" }), "ask: failed - timeout");
assert.strictEqual(p.progressLabel({ status: "custom" }), "custom");

assert.deepStrictEqual(p.resourceTargets([{ actionId: "a" }, { actionId: "" }, { actionId: "b" }]), ["a", "b"]);
assert.deepStrictEqual(p.trimLog([1, 2, 3], 2), [2, 3]);
assert.deepStrictEqual(p.trimLog([1], 2), [1]);
console.log("projectsctl ok");
