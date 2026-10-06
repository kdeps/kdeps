# Read a file in the REPL

`kdeps` with no arguments reads and writes files in the current directory. No workflow YAML. No API key. Install is the [30-second quickstart](/agent/quickstart). The file tools are [File tools](/agent/tools-files).

On the first run, kdeps asks you to confirm the `llama3.2:1b` download (about 1.1 GB). After that the model is cached in `~/.kdeps/models/`.

## Step 1: make a folder

```bash
mkdir notes-agent
cd notes-agent
cat > note.txt << 'EOF'
Ship the invoice by Friday.
EOF
```

## Step 2: start the REPL

```bash
kdeps
```

Run it from `notes-agent`. With no path, `kdeps` uses the current directory. A path argument loads a workflow, and this folder has none, so `kdeps .` exits with `no workflow or agency files found`.

```text
kdeps v2.x.x
model: llama3.2:1b (local, file backend)
> _
```

## Step 3: ask for the deadline

Paste this at the prompt:

```text
Read note.txt in this directory. Reply with the deadline in one sentence.
```

You get one sentence that names Friday. `note.txt` is unchanged. The read goes through `read_file`.

## Step 4: write the sentence back

Paste this next:

```text
Write that sentence to reply.txt.
```

You get `reply.txt` containing that sentence. Quit with `/exit`.

```bash
cat reply.txt
```
