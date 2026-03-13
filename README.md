# marp2pptx

Convert [Marp](https://marp.app/) Markdown presentations to PowerPoint (.pptx) files.

## Install

```bash
go install github.com/pascal/marp2pptx@latest
```

Or build from source:

```bash
make build
```

## Usage

```bash
marp2pptx presentation.md              # outputs presentation.pptx
marp2pptx presentation.md -o out.pptx  # custom output path
```

## Supported Markdown Features

- Headings (h1–h6)
- Paragraphs with **bold**, *italic*, `code`, and [links]()
- Unordered and ordered lists
- Fenced code blocks (rendered in Courier New)
- Tables (rendered as native PowerPoint tables)
- Images and background images (`![bg](image.jpg)`)
- Marp directives via HTML comments (`<!-- _backgroundColor: #264653 -->`)
- Slide splitting on `---`

## How It Works

The pipeline is: **Marp parser → Markdown converter → PPTX writer**

1. Parse Marp frontmatter and split slides on `---`
2. Convert each slide's markdown to an intermediate content model using [goldmark](https://github.com/yuin/goldmark)
3. Generate OOXML directly and write a valid `.pptx` ZIP archive — no external PowerPoint library required

## Development

```bash
make test           # run all tests
make test-verbose   # verbose test output
make lint           # vet + format check
make fmt            # format all Go files
make clean          # remove binary and generated pptx files
```

## License

MIT
