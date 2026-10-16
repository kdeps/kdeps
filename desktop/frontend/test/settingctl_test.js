"use strict";
const assert = require("assert");
const { settingControl, selectChoices } = require("../dist/settingctl.js");

assert.strictEqual(settingControl({ type: "bool" }), "checkbox");
assert.strictEqual(settingControl({ type: "string", secret: true, options: ["a"] }), "password");
assert.strictEqual(settingControl({ type: "string", options: ["a", "b"] }), "select");
assert.strictEqual(settingControl({ type: "number", min: 0, max: 2, step: 0.1 }), "range");
assert.strictEqual(settingControl({ type: "number", min: 0 }), "number");
assert.strictEqual(settingControl({ type: "string", suggestions: ["30s"] }), "suggest");
assert.strictEqual(settingControl({ type: "string", path: "defaults.timezone" }), "suggest");
assert.strictEqual(settingControl({ type: "string", path: "x" }), "text");
assert.strictEqual(settingControl({ type: "int", path: "x" }), "number");

assert.deepStrictEqual(selectChoices({ options: ["a", "b"], value: "a" }), ["", "a", "b"]);
assert.deepStrictEqual(selectChoices({ options: ["a"], value: "zzz" }), ["", "a", "zzz"]);
assert.deepStrictEqual(selectChoices({ options: ["a"], value: null }), ["", "a"]);
console.log("settingctl ok");
