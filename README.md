## What is Ghost

Ghost is a command-line interface (CLI) that integrates with multiple Large Language Models (LLMs) and utilizes the Model Context Protocol (MCP).

By combining LLMs and MCP, users can leverage Ghost to perform various tasks. The core idea is to use the capabilities of LLMs to interact with MCPs to accomplish specific goals.

Potential Scenarios:
- **Coding:** This can be achieved by using MCPs that interact with tools like Git, databases, GitHub, etc.
- **Chat** Just console chat.

### Name

I am planning to develop a Go MCP host, simplified to Go host, which is Ghost.
I think this is a good name. Considering `Ghost in the Shell`, it feels like a suitable name for the age of AI.

## Usage

**Install**:
```bash
git clone git@github.com:kk2simon/ghost-cli.git
cd ghost-cli
go install ./ghost
```

**Example Config**:
```toml
[[LLMs]]
Name = "openai"
APIType = "openaichat" #use openai chat api
APIKey = "$your_openai_api_key"
Model = "gpt-4.1-mini"

[[LLMs]]
Name = "gemini"
APIType = "gemini" #use google gemini api
APIKey = "$your_gemini_api_key"
Model = "models/gemini-2.5-flash-preview-04-17"

[[Mcps]]
Name = "git"
Command = "uvx"
Env = []
Args = ["mcp-server-git", "--repository", "/home/foo/bar/workspace"]

[[Mcps]]
Name = "filesystem"
Command = "npx"
Env = []
Args = [
    "-y",
    "@modelcontextprotocol/server-filesystem",
    "/home/foo/bar/workspace",
]

```

**Example prompt**: [examples/coding/prompt-coding.md](examples/coding/prompt-coding.md)

**Usage**:
```bash
# -c config.toml : config file path
# -p prompt.md : prompt file path
# -l openai : LLM name, configured in config.toml
# -m gpt-4.1-mini : default use configured model in config.toml
ghost -c ./ghost/config.toml -l gemini -p ./.ghost/prompt-coding.md 
```

**Prompt Template**:

Now, support `{{.cwd}}`, `{{.dirTree}}` placeholder.

**Suggestions**:
- only use mcp that needed
- DON'T put sensitive data working directory, this avoid read by LLM
- Built-in prompt template support `{{.dirTree}}` it ignore files in `.gitignore`
  - If you don't want AI to read all files, you should tell AI in prompt (don't use some specific tools)


## Workflow

1. The user inputs a command via the CLI.
2. Ghost gathers context, constructs the LLM prompt, and calls the LLM API.
3. Ghost waits for the LLM's response and calls the appropriate MCPs as required by the LLM, continuing until no further MCP calls are needed.

## Features

- Support for multiple LLMs.
- Configurable MCPs.
- Prompt templates.

## Roadmap
- [ ] Support for SSE/streamed output.
- [ ] `--var "foo=bar"` support, this allows customize template context vars.
- [ ] System/developer prompt, planning to parse from prompt markdown file.
- [ ] Flags to auto approve tool use.
