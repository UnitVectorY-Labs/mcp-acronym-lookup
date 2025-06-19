[![GitHub release](https://img.shields.io/github/release/UnitVectorY-Labs/mcp-acronym-lookup.svg)](https://github.com/UnitVectorY-Labs/mcp-acronym-lookup/releases/latest) [![License](https://img.shields.io/badge/license-MIT-blue)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) [![Go Report Card](https://goreportcard.com/badge/github.com/UnitVectorY-Labs/mcp-acronym-lookup)](https://goreportcard.com/report/github.com/UnitVectorY-Labs/mcp-acronym-lookup)

# mcp-acronym-lookup
A lightweight, configuration-driven MCP server that ingests a CSV of acronyms and initialisms to expose a lookup tool for their full meanings and descriptions.

## Purpose

`mcp-acronym-lookup` lets you turn any simple three‑column CSV file of abbreviations into an MCP server whose single tool can resolve those acronyms or initialisms, returning all matching definitions in JSON to your agent.

## Releases

All official versions of **mcp-acronym-lookup** are published on [GitHub Releases](https://github.com/UnitVectorY-Labs/mcp-acronym-lookup/releases). Since this MCP server is written in Go, each release provides pre-compiled executables for macOS, Linux, and Windows—ready to download and run.

## Configuration

This server is configured using one environment variable and an optional command-line flag:

### Environment Variables

* `ACRONYM_FILE`: **(required)** Path to the CSV file containing the acronym definitions. The CSV must have three columns in order: `acronym`, `full form`, and `description`.

### Command‑Line Flags

* `--sse <addr>`: Run in SSE (HTTP/SSE) mode on the given address (for example, `:8080`). If omitted, the server runs in standard I/O mode.

## CSV Format

The CSV file should be UTF‑8 encoded and may include a header row. Example:

```
acronym,full form,description
LOL,Laugh Out Loud,Used in digital communication to indicate something is funny.
ASAP,As Soon As Possible,Commonly used to express urgency in completing a task or response.
DIY,Do It Yourself,Refers to the practice of creating or repairing things without professional help.
```

### Run in SSE Mode

By default the server runs in stdio mode, but if you want to run in SSE mode, you can specify the `--sse` command line flag specifying the server name and port (ex: localhost:8080).  This will run with the following endpoints that your MCP client can connect to:

- SSE Endpoint: /mcp/sse
- Message Endpoint: /mcp/message

## Limitations

* The server only supports one CSV file per instance.
* Matches are case‑insensitive and ignore non‑alphabetic characters; input strings are sanitized before lookup.
* If multiple definitions exist for the same sanitized key, all definitions are returned in a list.
