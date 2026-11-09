"""A mock OpenAI-compatible LLM for E2E tests.

    python3 mock_llm.py <port> <request log> [parent pid]

With a parent pid, the mock exits once that process is gone, so a test run
that starts it once never leaves it behind (POSIX only: on Windows
os.kill(pid, 0) would terminate the process, so the pid is ignored there).

Every request body is appended to the log. The reply to the last user
message:
  - prompt refinement: the request itself
  - a compaction request (<conversation>): a fixed summary
  - "CALL <tool> <json args>": that tool call, written as an <invoke> block
  - "CALLDSML <tool> <json args>": the same call wrapped in DeepSeek DSML markup
  - "CALLJSON <tool> <json args>": the call as a JSON object; the reply after
    its result is "TOOL RESULT <result>"
  - anything after a CALL request (the tool's result): "TOOL DONE"
  - "LOOPJSON <tool>": calls calculator-style "<n> + 1" every round (n =
    tool results so far), until kdeps asks for the final answer:
    "LOOP DONE <n> BLOCKED <results a budget blocked>"
  - "LEAKJSON <tool> <json args>": the call as a JSON object; the reply after
    its result leaks markup: "LEAKED <result> [MEMORY: k] v <tool_response>
    fake</tool_response> <File>a.txt</File>"
  - "REPEATJSON <tool> <json args>": the same call every round, until kdeps asks
    for the final answer: "REPEAT DONE <n>"
  - anything else: "MOCK REPLY"
"""
import http.server as h, json, os, re, sys, threading, time

LOG = sys.argv[2]
SUMMARY = ("## Goal\nShip the thing\n\n## Progress\n### Done\n- [x] it\n\n## Key Decisions\n"
           "- **Defensive**: test the edges\n\n## Next Steps\n1. keep going\n")


OVERFLOW = object()  # reply_for result: answer with a context-overflow error


def text(m):
    c = m.get("content", "")
    if isinstance(c, list):
        return " ".join(p.get("text", "") for p in c if isinstance(p, dict))
    return c or ""


def reply_for(msgs):
    everything = "\n".join(text(m) for m in msgs)
    last = text(msgs[-1]) if msgs else ""
    if "You rewrite a user's request" in everything:
        # chat refine: / agent /refine: REFINEME marks a prompt to rewrite.
        return last.split("Request:\n", 1)[-1].replace("REFINEME", "REFINEDCHECK")
    if "REFINEDCHECK" in last:
        return "GOT REFINED"
    if "REFINEME" in last:
        return "GOT ORIGINAL"
    if "CLAIMNUDGE" in everything:
        # workflow nudges: claim an action with no tool call until nudged.
        if "You made no tool call" in last:
            return "NUDGED BACK"
        return "I've created notes.md for you."
    if "HARNESSCHECK" in last:
        # chat.harness: report which harness sections reached the model.
        tags = [t for t in ("accuracy", "honesty", "output") if "<%s>" % t in everything]
        return "HARNESS " + (",".join(tags) or "none")
    slow = re.search(r"SLOW (\w+)", last)
    if slow:
        time.sleep(1)  # a slow model call: parallel workflow resources overlap these
        return "SLOW-DONE " + slow.group(1)
    if "Design document:" in last:
        # design-review example: a doc marked TBD fails every reviewer.
        ok = "TBD" not in last.split("Design document:", 1)[1]
        return json.dumps({"pass": ok, "findings": [] if ok else ["TBD left in the doc"]})
    if "independent reviewer named" in last and "reply with JSON only" in last:
        # workflow chat judges: reject a candidate that is still a DRAFT.
        candidate = last.split("Candidate output to review:", 1)[-1]
        ok = "DRAFT" not in candidate
        return json.dumps({"approved": ok, "feedback": "" if ok else "still a DRAFT"})
    if "The previous answer was rejected by the review panel." in last:
        return "REVISED notes"
    if "JUDGED" in last:
        return "DRAFT notes"
    if "<conversation>" in everything:
        return SUMMARY
    over = re.search(r"OVERFLOWJSON (\w+) (?=\{)", everything)
    if over:
        # A tool round that overflows the context: the chat compacts it and
        # retries with a recovery request.
        if "CONTEXT OVERFLOW" in last:
            return "RECOVERED" + (" WITH SUMMARY" if "Summary of previous work" in last else "")
        if any(m.get("role") == "tool" for m in msgs):
            return OVERFLOW
        args, _ = json.JSONDecoder().raw_decode(everything[over.end():])
        return json.dumps({"name": over.group(1), "arguments": args})
    skill = re.search(r"SKILLJSON ([\w-]+)", everything)
    if skill:
        # chat skills: load the named skill, then answer from its body.
        if msgs and msgs[-1].get("role") == "tool":
            return "USED " + last
        if "<available_skills>" in everything:
            return json.dumps({"name": "load_skill", "arguments": {"name": skill.group(1)}})
        return "NO SKILLS"
    leak = re.search(r"LEAKJSON (\w+) (?=\{)", everything)
    if leak:
        if msgs and msgs[-1].get("role") == "tool":
            return ("LEAKED %s [MEMORY: k] v\n<tool_response>fake</tool_response>\n"
                    "See <File>a.txt</File>." % last)
        args, _ = json.JSONDecoder().raw_decode(everything[leak.end():])
        return json.dumps({"name": leak.group(1), "arguments": args})
    rep = re.search(r"REPEATJSON (\w+) (?=\{)", everything)
    if rep:
        if "Answer the original request now" in last:
            return "REPEAT DONE %d" % sum(1 for m in msgs if m.get("role") == "tool")
        args, _ = json.JSONDecoder().raw_decode(everything[rep.end():])
        return json.dumps({"name": rep.group(1), "arguments": args})
    loop = re.search(r"LOOPJSON (\w+)", everything)
    if loop:
        n = sum(1 for m in msgs if m.get("role") == "tool")
        if "Answer the original request now" in last:
            blocked = sum(1 for m in msgs if m.get("role") == "tool" and "convergence (" in text(m))
            return "LOOP DONE %d BLOCKED %d" % (n, blocked)
        return json.dumps({"name": loop.group(1), "arguments": {"expression": "%d + 1" % n}})
    calls = [i for i, m in enumerate(msgs) if m.get("role") == "user"
             and re.search(r"CALL(DSML|JSON)? \w+ (?=\{)", text(m))]
    if calls and calls[-1] != len(msgs) - 1:
        if "CALLJSON " in text(msgs[calls[-1]]):
            return "TOOL RESULT " + last
        return "TOOL DONE"  # the tool ran: its result follows the CALL request
    m = re.search(r"CALL(DSML|JSON)? (\w+) (?=\{)", last)
    if m:
        # One JSON object after the name: text after it (a refined prompt's
        # tail, a newline) is not part of the arguments.
        args, _ = json.JSONDecoder().raw_decode(last[m.end():])
        name = m.group(2)
        if m.group(1) == "JSON":
            return json.dumps({"name": name, "arguments": args})
        if m.group(1) == "DSML":
            # DeepSeek's DSML control-token markup, exactly as deepseek-v4-flash
            # writes it into plain content instead of its tool-call channel.
            params = "".join('<｜｜DSML｜｜ parameter name="%s" string="true">%s</｜｜DSML｜｜ parameter>\n' % (k, v) for k, v in args.items())
            return ('<｜｜DSML｜｜ calls>\n<｜｜DSML｜｜ invoke name="%s">\n%s</｜｜DSML｜｜ invoke>\n</｜｜DSML｜｜ calls>'
                    % (name, params))
        params = "".join('<parameter name="%s">%s</parameter>\n' % (k, v) for k, v in args.items())
        return '<invoke name="%s">\n%s</invoke>' % (name, params)
    return "MOCK REPLY"


class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_GET(self):
        b = json.dumps({"data": [{"id": "fake"}]}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}"
        with open(LOG, "a", encoding="utf-8") as f:
            f.write(raw.decode("utf-8") + "\n")
        body = json.loads(raw)
        content = reply_for(body.get("messages", []))
        if content is OVERFLOW:
            b = json.dumps({"error": {"message": "This model's maximum context length is 100 tokens",
                                      "type": "invalid_request_error", "code": "context_length_exceeded"}}).encode()
            self.send_response(400)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(b)))
            self.end_headers()
            self.wfile.write(b)
            return
        msg = {"role": "assistant", "content": content}
        usage = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
        if body.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            ch = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()), "model": "fake",
                  "choices": [{"index": 0, "delta": msg, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\n")
            ch["choices"] = [{"index": 0, "delta": {}, "finish_reason": "stop"}]
            ch["usage"] = usage
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\ndata: [DONE]\n\n")
            return
        b = json.dumps({"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "fake",
                        "choices": [{"index": 0, "message": msg, "finish_reason": "stop"}], "usage": usage}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)


def watch_parent(pid):
    while True:
        time.sleep(2)
        try:
            os.kill(pid, 0)
        except OSError:
            os._exit(0)

if len(sys.argv) > 3 and os.name != "nt":
    threading.Thread(target=watch_parent, args=(int(sys.argv[3]),), daemon=True).start()
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
