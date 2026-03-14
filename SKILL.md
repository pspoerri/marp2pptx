---
name: marp2pptx
description: Convert Marp markdown slides to PowerPoint (.pptx) using the marp2pptx CLI tool
---

## What I do

Convert Marp-formatted markdown files into PowerPoint (.pptx) presentations using the `marp2pptx` command-line tool.

## When to use me

Use this skill when the user wants to export or convert a Marp markdown presentation to PowerPoint format (.pptx).

## How to use

Run the `marp2pptx` CLI tool via Bash:

```bash
marp2pptx -o <output.pptx> <input.md>
```

### Flags

- `-o <path>` -- specify the output .pptx file path
- `-version` -- print the tool version

### Example

For this project, to convert `slides.md` to PowerPoint:

```bash
marp2pptx -o slides.pptx slides.md
```

### Notes

- The input file must be a valid Marp markdown file (with `marp: true` in the frontmatter).
- If no `-o` flag is provided, the tool will use a default output path.
- This is an alternative to the built-in `marp-cli` PPTX export (`npm run build:pptx`), which uses a different conversion approach.
