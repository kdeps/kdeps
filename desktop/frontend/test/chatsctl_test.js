"use strict";
const assert = require("assert");
const { folderGroups, folderName } = require("../dist/chatsctl.js");

const hit = (id, cwd) => ({ session: { id, cwd } });
const groups = folderGroups([hit("1", "/a/web"), hit("2", "/b/api"), hit("3", "/a/web")], "/b/api");
assert.deepStrictEqual(groups.map((g) => g.dir), ["/a/web", "/b/api"], "folder of the newest chat first");
assert.deepStrictEqual(groups[0].hits.map((h) => h.session.id), ["1", "3"]);
assert.strictEqual(groups[0].name, "web");
assert.strictEqual(groups[0].current, false);
assert.strictEqual(groups[1].current, true);
assert.deepStrictEqual(folderGroups(null, "/x"), []);

assert.strictEqual(folderName("C:\\Users\\me\\proj"), "proj");
assert.strictEqual(folderName("/home/me/proj/"), "proj");
assert.strictEqual(folderName(""), "?");
console.log("chatsctl ok");
