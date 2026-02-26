# DebateGPT

**DebateGPT** is a terminal-based CLI where multiple LLM agents (Claude, GPT-4o, Gemini, and more) **debate** each other across rounds to produce more accurate, well-reasoned answers. Built for the brainstorming and architecture phase — before writing any code.

Instead of relying on a single model's response, DebateGPT orchestrates a structured debate: agents respond independently, critique each other, defend or concede points, and a configurable judge synthesizes everything into an actionable brainstorm document.

## Why DebateGPT?

- **Reduces bias** — Multiple models challenge each other's assumptions and blind spots
- **Improves accuracy** — Cross-critique filters out weak or incorrect answers
- **Produces richer responses** — The final answer synthesizes the best points from all participants
- **Configurable perspectives** — Assign roles like Pragmatist, Architect, or Contrarian to force genuine tension

## How It Works

1. **Prompt** — Submit a question to DebateGPT
2. **Debate** — Multiple agents independently generate responses in parallel (streaming)
3. **Critique** — Each agent reviews and critiques the others using a structured format
4. **Defend/Concede** — Agents explicitly state what they changed their mind on
5. **Synthesis** — A configurable judge produces a final brainstorm document

---

## Installation

### Prerequisites

- **Go 1.21+** — [Install Go](https://go.dev/dl/)
- API keys for at least **2** of the following:
  - [OpenAI](https://platform.openai.com/api-keys) (GPT-4o)
  - [Anthropic](https://console.anthropic.com/) (Claude)
  - [Google AI](https://aistudio.google.com/apikey) (Gemini)

### Build from Source

```bash
git clone https://github.com/aadidevcb/DebateGPT.git
cd DebateGPT
go build -o debategpt .
```

Optionally, install it globally:

```bash
go install .
```

### Configuration

1. **Create the config directory and file:**

```bash
mkdir -p ~/.debategpt
cp config.example.yaml ~/.debategpt/config.yaml
```

2. **Set your API keys** (choose one method):

**Option A — Environment variables (recommended):**

```bash
# Add to your ~/.zshrc or ~/.bashrc
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export GEMINI_API_KEY="AI..."
```

The config file references these via `${ANTHROPIC_API_KEY}` syntax.

**Option B — Directly in config:**

```bash
debategpt config set agents.claude.api_key "sk-ant-..."
debategpt config set agents.openai.api_key "sk-..."
debategpt config set agents.gemini.api_key "AI..."
```

**Option C — Edit `~/.debategpt/config.yaml` directly:**

```yaml
agents:
  claude:
    model: claude-sonnet-4-20250514
    api_key: sk-ant-your-key-here
  openai:
    model: gpt-4o
    api_key: sk-your-key-here
  gemini:
    model: gemini-2.5-pro
    api_key: AI-your-key-here
```

### Verify Installation

```bash
debategpt version
# DebateGPT v0.1.0

debategpt --help
```

---

## Usage

### Basic Debate

```bash
debategpt debate "How should I design a microservices auth system?"
```

This will:
1. Load agents from your config
2. Check for `DEBATE.md` in the current directory
3. Run 2 rounds of debate (default) with streaming output
4. Judge synthesizes a final document
5. Save the result to `./brainstorms/`

### With Options

```bash
debategpt debate \
  --agents claude,openai,gemini \
  --rounds 3 \
  --judge separate \
  --budget 0.50 \
  --output ./brainstorms/auth-design.md \
  "How should I design the auth system?"
```

### Quick Mode

1 round, cheaper models — for when you want a fast answer:

```bash
debategpt debate --quick "Should I use REST or GraphQL?"
```

### Context Injection

Feed project files to the agents for project-aware brainstorming:

```bash
debategpt debate \
  --context ./README.md,./schema.sql,./go.mod \
  "How should I design the API layer?"
```

### Judge Modes

```bash
# Cheapest — one debater doubles as judge
debategpt debate --judge participant "..."

# Most objective — dedicated model synthesizes (default)
debategpt debate --judge separate "..."

# Most thorough — all agents summarize, then merge
debategpt debate --judge consensus "..."
```

### Include Full Transcript

```bash
debategpt debate --transcript -o design.md "..."
```

### All Flags

| Flag | Description |
|------|-------------|
| `--agents` | Comma-separated agents to use (e.g., `claude,openai,gemini`) |
| `--rounds` | Number of debate rounds (default: 2) |
| `--judge` | Judge mode: `participant`, `separate`, `consensus` |
| `--budget` | Max budget in USD (0 = unlimited) |
| `-o, --output` | Output file path |
| `--quick` | Quick mode (1 round) |
| `--transcript` | Include full debate transcript in output |
| `--context` | Comma-separated context files to include |

---

## DEBATE.md — Project-Level Configuration

Inspired by `CLAUDE.md`, you can create a `DEBATE.md` file in your project to customize debate behavior. DebateGPT walks up from your current directory and merges all `DEBATE.md` files found (like `.gitignore`).

### Initialize

```bash
debategpt init
# ✅ Created DEBATE.md in current directory
```

### Example DEBATE.md

```markdown
# Debate Rules

## Context
This is a Go microservices project using gRPC, PostgreSQL, and Redis.
We prioritize reliability over speed-to-market.

## Perspectives
- pragmatist: "Optimize for Go idioms, stdlib where possible, minimize dependencies"
- architect: "Design for horizontal scale, think about the 10x growth case"
- security: "Assume adversarial inputs, find every auth/authz gap"

## Constraints
- Must be deployable on Kubernetes
- No suggestions involving MongoDB (we've committed to PostgreSQL)

## Debate Style
rounds: 3
judge: separate
critique_format: structured
temperature: 0.8

## Focus Areas
- Error handling patterns
- API versioning strategy
```

### Resolution Order

```
~/.debategpt/DEBATE.md          ← global defaults
~/CODE/myproject/DEBATE.md      ← project overrides
~/CODE/myproject/api/DEBATE.md  ← subdirectory specialization
```

---

## Adding Custom Agents

Any **OpenAI-compatible API** can be added via config — no code changes needed:

```yaml
agents:
  # Local Ollama
  llama-local:
    provider: openai-compatible
    base_url: http://localhost:11434/v1
    model: llama3.2
    api_key: ollama

  # DeepSeek
  deepseek:
    provider: openai-compatible
    base_url: https://api.deepseek.com/v1
    model: deepseek-chat
    api_key: ${DEEPSEEK_API_KEY}

  # Mistral
  mistral:
    provider: openai-compatible
    base_url: https://api.mistral.ai/v1
    model: mistral-large-latest
    api_key: ${MISTRAL_API_KEY}

  # Groq
  groq:
    provider: openai-compatible
    base_url: https://api.groq.com/openai/v1
    model: llama-3.3-70b-versatile
    api_key: ${GROQ_API_KEY}
```

Then use them:

```bash
debategpt debate --agents claude,deepseek,llama-local "..."
```

---

## Output

DebateGPT generates structured markdown brainstorm documents:

```markdown
# Brainstorm: How should I design the auth system?
*DebateGPT | 3 agents, 2 rounds | Judge: separate | Cost: $0.215*

## Executive Summary
## Recommended Approach
## Key Decision Points
## Risks & Mitigations
## Points of Consensus
## Unresolved Disagreements
## Action Items
```

Output is saved to `./brainstorms/` by default, or specify with `--output`.

---

## Cost Management

DebateGPT tracks token usage in real-time and displays a cost summary after each debate:

```
╭──────────────────── Cost Summary ────────────────────╮
│ Agent       Round 1    Round 2    Judge     Total     │
│ Claude      $0.032     $0.041     —         $0.073    │
│ GPT-4o      $0.028     $0.035     —         $0.063    │
│ Gemini      $0.015     $0.019     $0.045    $0.079    │
│──────────────────────────────────────────────────────│
│ Total                                       $0.215    │
╰──────────────────────────────────────────────────────╯
```

Set a budget to prevent surprises:

```bash
debategpt debate --budget 0.50 "..."
```

---

## Use Cases

- Architecture brainstorming before writing code
- Research assistance requiring balanced perspectives
- Complex problem-solving where multiple approaches matter
- Decision support with pros/cons analysis
- API design review and trade-off analysis

## License

This project is licensed under the [MIT License](LICENSE).

---

<p align="center">Made with ❤️ by <a href="https://www.linkedin.com/in/aadidevcb/">Aadidev</a></p>
