# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

marp2pptx — a Go CLI tool that converts Marp Markdown presentations to PowerPoint (.pptx) format.

## Commands

```bash
make build          # Build binary
make test           # Run all tests
make test-verbose   # Run tests with verbose output
make test-run RUN=TestName  # Run a single test by name
make lint           # Run vet + format check
make fmt            # Format all Go files
make clean          # Remove binary and generated pptx files
make run ARGS="input.md -o output.pptx"  # Build and run
```

## Architecture

Pipeline: **Marp parser → Markdown converter → PPTX writer**

- `main.go` — CLI entry point, orchestrates the pipeline
- `internal/marp/` — Parses Marp documents: YAML frontmatter extraction, slide splitting on `---`, HTML comment directive extraction (`<!-- _class: lead -->`)
- `internal/markdown/` — Converts slide markdown to an intermediate representation (`ContentBlock`/`Run` types) using goldmark. Handles headings, paragraphs, lists, code blocks, images, and inline formatting (bold/italic/code/links)
- `internal/pptx/` — Generates PPTX files by writing OOXML directly (no external PPTX library). Assembles ZIP with `encoding/xml` + `archive/zip`. Static parts (theme, slide master, layout) are boilerplate; slides are dynamically generated from content blocks

### Key design decisions

- **Custom PPTX writer** instead of unioffice (commercial license) — the OOXML subset needed is small enough to emit directly
- **Intermediate content model** (`ContentBlock`/`Run`) decouples markdown parsing from PPTX generation
- **Standard `flag`** for CLI — single-command tool, no need for cobra

### Dependencies

- `github.com/yuin/goldmark` — CommonMark markdown parser
- `gopkg.in/yaml.v3` — YAML frontmatter parsing

## Instructions

- When referencing Go libraries from external git repositories, clone the whole repo into a local cache rather than fetching individual files
