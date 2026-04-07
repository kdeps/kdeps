# LLM Chat with Tools

Interactive LLM chat (stdin REPL) with **10 built-in tools**. The LLM can call any tool
automatically when it needs factual data or computation.

## Tools Available

| Tool | Description |
|------|-------------|
| `calculate` | Math: arithmetic, functions (sqrt, sin, log, factorial...), constants (pi, e) |
| `get_weather` | Current weather for any city (mock — connect real API for production) |
| `get_time` | Current date/time for a timezone (UTC, EST, PST, CET, IST, JST...) |
| `convert_units` | Length, weight, speed, volume, area, temperature, data sizes |
| `analyze_text` | Word/sentence count, readability score, top words, reading time |
| `format_json` | Validate and pretty-print JSON, optional key sorting |
| `base64` | Encode or decode Base64 (standard or URL-safe) |
| `parse_url` | Parse URL components or encode/decode URL strings |
| `hash` | MD5, SHA1, SHA256, SHA512, SHA3, BLAKE2 hashes and HMACs |
| `random` | Numbers, floats, passwords, UUIDs, hex tokens, dice, coin, choices |

## Run

```bash
kdeps run workflow.yaml --dev
```

The chat starts an interactive REPL:

```
You: What is the SHA256 hash of "hello world"?
Assistant: The SHA256 hash of "hello world" is b94d27b9...

You: Convert 100 km to miles
Assistant: 100 km = 62.137119 miles

You: Roll 3d6
Assistant: Your 3d6 rolls: [4, 2, 6] — Total: 12

You: /quit
```

Type `/quit` or `/exit` to end the session.

## Example Prompts

```
What is sqrt(2) raised to the power of 10?
What's the weather in Tokyo?
What time is it in IST right now?
Convert 5 pounds to kilograms
Analyze this text: The quick brown fox jumps over the lazy dog.
Format this JSON: {"name":"Alice","age":30}
Base64 encode "Hello, KDeps!"
Parse https://example.com/search?q=kdeps&page=2
Hash "password123" using bcrypt — wait, use SHA256
Generate 5 random UUIDs
Pick a random item from: pizza,sushi,tacos,burger
```

## Structure

```
llm-chat-tools/
|- workflow.yaml           # sources: [llm], executionType: stdin
+- resources/
   |- 01-calculator.yaml   # calcTool
   |- 02-weather.yaml      # weatherTool
   |- 03-time.yaml         # timeTool
   |- 04-unit-converter.yaml  # unitConverterTool
   |- 05-text-analyzer.yaml   # textAnalyzerTool
   |- 06-json-formatter.yaml  # jsonFormatterTool
   |- 07-base64.yaml          # base64Tool
   |- 08-url-parser.yaml      # urlParserTool
   |- 09-hash.yaml            # hashTool
   |- 10-random.yaml          # randomTool
   +- 11-chat.yaml            # main chat resource with all tools
```

## API Server Mode

To expose as an HTTP endpoint instead of stdin REPL, change `executionType` in `workflow.yaml`:

```yaml
settings:
  input:
    llm:
      executionType: apiServer
```

Then query:

```bash
curl -X POST 'http://localhost:16401/api/v1/llm-chat-tools?message=What+is+pi+times+2'
```
