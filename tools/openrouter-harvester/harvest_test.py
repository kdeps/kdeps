#!/usr/bin/env python3
"""Filter and version-bump tests for the OpenRouter free-model harvest."""

import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import harvest  # noqa: E402


def model(mid, prompt="0", completion="0", outputs=None, params=None, inputs=None,
          ctx=8192, max_out=1024, name=None, request=None):
    pricing = {"prompt": prompt, "completion": completion}
    if request is not None:
        pricing["request"] = request
    body = {
        "id": mid,
        "name": name or mid,
        "pricing": pricing,
        "context_length": ctx,
        "top_provider": {"max_completion_tokens": max_out},
        "supported_parameters": params or [],
        "architecture": {
            "input_modalities": inputs or ["text"],
            "output_modalities": ["text"] if outputs is None else outputs,
        },
    }
    return body


class FreeModelsTest(unittest.TestCase):
    def test_keeps_zero_price_text_models_sorted(self):
        payload = {"data": [
            model("b/b:free", params=["reasoning"], inputs=["text", "image"], name="B  model"),
            model("a/a:free"),
            model("paid/x", prompt="0.000001"),
            model("song/x", outputs=["text", "audio"]),
            model("img/x", outputs=["image"]),
            model("norate/x", prompt=None, completion="0"),
            model("percall/x", request="0.01"),
            model("openrouter/free", name="Free Models Router"),
            model("nocontext/x", ctx=0),
            model("noout/x", max_out=0),
        ]}
        got = harvest.free_entries(payload)
        self.assertEqual([e["id"] for e in got], ["a/a:free", "b/b:free", "openrouter/free"])
        b = got[1]
        self.assertEqual(b["desc"], "B model")
        self.assertTrue(b["thinking"])
        self.assertTrue(b["images"])
        self.assertEqual(b["backend"], "openrouter")
        self.assertEqual(b["envVar"], "OPENROUTER_API_KEY")
        self.assertFalse(got[0]["thinking"])
        self.assertFalse(got[0]["images"])

    def test_missing_output_modalities_counts_as_text(self):
        raw = model("plain/x")
        del raw["architecture"]["output_modalities"]
        self.assertTrue(harvest.is_free_model(raw))

    def test_rejects_a_payload_without_data(self):
        with self.assertRaises(ValueError):
            harvest.free_entries({"models": []})

    def test_version_bump(self):
        one = harvest.free_entries({"data": [model("a/a:free")]})
        two = harvest.free_entries({"data": [model("a/a:free"), model("b/b:free")]})
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "openrouter.yaml")
            first = harvest.registry_text(path, one)
            self.assertIn("version: 1.0.0\n", first)
            with open(path, "w", encoding="utf-8") as f:
                f.write(first)
            same = harvest.registry_text(path, one)
            self.assertIn("version: 1.0.0\n", same)
            changed = harvest.registry_text(path, two)
            self.assertIn("version: 1.0.1\n", changed)
            self.assertIn("# Entries: 2\n", changed)


class ParameterCountTest(unittest.TestCase):
    def test_largest_count_in_id(self):
        import harvest

        self.assertEqual(550.0, harvest.parameter_count("nvidia/nemotron-3-ultra-550b-a55b:free"))
        self.assertEqual(2.6, harvest.parameter_count("liquid/lfm-2.5-2.6b:free"))
        self.assertEqual(0, harvest.parameter_count("poolside/laguna-xs-2.1:free"))

    def test_row_writes_params_only_when_known(self):
        import harvest

        row = "\n".join(harvest.entry_lines({
            "id": "x/y-31b:free", "desc": "d", "thinking": False, "images": False,
            "contextWindow": 1, "maxOutputTokens": 2, "params": 31,
        }))
        self.assertIn("    params: 31\n", row + "\n")


if __name__ == "__main__":
    unittest.main()
