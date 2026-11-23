# Spun Records and Lounge

An experiment in open-sourcing a business: a record store plus a listening lounge. The code, the plans, and the
operating documents all live in this repository.

The repo includes `spun`, a command-line tool that renders a directory of Markdown documents (a *bundle*) into a single
PDF. It is how the business plan and operating documents are published.

## Prerequisites

- **Go**: version 1.27 or newer, as declared in [`go.mod`](go.mod).
- **Chromium**: `spun` prints PDFs through a headless Chromium (or Chrome). It looks for the executable in this order:
  1. the `--chromium <path>` flag,
  2. the `SPUN_CHROMIUM` environment variable,
  3. `chromium`, `chromium-browser`, `google-chrome`, and `google-chrome-stable` on your `PATH`.

  ```sh
  export SPUN_CHROMIUM=/path/to/chromium
  ```

## Building and installing

```sh
go build ./cmd/spun      # writes ./spun
go install ./cmd/spun    # installs spun into $(go env GOPATH)/bin
```

You can also run it without building: `go run ./cmd/spun --help`.

## Usage

```sh
spun docs render <bundle-dir> [-o out.pdf] [--chromium path]
```

| Flag               | Meaning                                                                                           |
| ------------------ | ------------------------------------------------------------------------------------------------- |
| `-o`, `--output`   | Output PDF path. Overrides the bundle's `output`; relative paths resolve against the working directory. |
| `--chromium`       | Path to the Chromium executable. Overrides `SPUN_CHROMIUM` and the `PATH` search.                 |

The command prints the path of the PDF it wrote. Example, using the small bundle the tests use:

```sh
go run ./cmd/spun docs render internal/cli/testdata/minimal-bundle -o /tmp/readme-check.pdf
go run ./cmd/spun docs render internal/cli/testdata/minimal-bundle -o /tmp/readme-check.pdf --chromium /path/to/chromium
```

## Bundle layout

A bundle is a directory containing a `spun.yaml` and the Markdown documents it lists, plus any images they use:

```text
my-bundle/
├── spun.yaml
├── intro.md
├── details.md
└── images/
    └── floor-plan.png
```

Documents are concatenated into one PDF in the order listed under `documents`.

## `spun.yaml` reference

Only `title` and `documents` are required. Unknown keys are rejected.

```yaml
title: Spun Records and Lounge
subtitle: Business plan
author: The Spun team
date: 2026-11-23
documents:
  - intro.md
  - details.md
template: default
output: plan.pdf
page:
  size: letter
  margins: 1in
  numbers: true
numbering:
  enabled: true
  depth: 3
features:
  math: true
```

| Key                  | Required | Default                    | Description                                                                                                  |
| -------------------- | -------- | -------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `title`              | yes      |                            | Document title, shown on the cover.                                                                          |
| `subtitle`           | no       | empty                      | Shown under the title when set.                                                                              |
| `author`             | no       | empty                      | Shown on the cover when set.                                                                                 |
| `date`               | no       | empty                      | Free-form text shown on the cover when set; it is not parsed or generated.                                   |
| `documents`          | yes      |                            | Ordered list of Markdown files, relative to the bundle. Each must exist inside the bundle and be listed once. |
| `template`           | no       | `default`                  | `default` for the built-in template, or a path inside the bundle to an `.html` file or to a directory holding `template.html` and optionally `style.css`. |
| `output`             | no       | `<bundle-dir-name>.pdf`    | Output PDF path, relative to the bundle directory unless absolute. The `-o` flag overrides it.               |
| `page.size`          | no       | `letter`                   | `letter` or `a4`.                                                                                            |
| `page.margins`       | no       | `1in`                      | Any CSS margin value, such as `1in` or `20mm 15mm`. Must not be empty.                                       |
| `page.numbers`       | no       | `true`                     | Print "page / total" in the footer. Also controls the page numbers in cross-references.                      |
| `numbering.enabled`  | no       | `true`                     | Number headings (`1`, `1.1`, `1.1.1`, …).                                                                    |
| `numbering.depth`    | no       | `3`                        | Deepest heading level that is numbered, from `1` to `6`.                                                     |
| `features`           | no       | none enabled               | Map of optional feature names to a bool, or to a mapping with an optional `enabled` (default `true`) plus feature options. Unknown feature names are rejected. |

The one available feature is `math`:

```yaml
features:
  math: true          # shorthand
  # or
  math:
    enabled: true
```

## Authoring

Documents are CommonMark Markdown with tables and the extensions below.

### Math

With `features: {math: true}`, TeX between dollar signs is typeset with KaTeX (bundled, so rendering works offline):

- Inline: `$a^2 + b^2 = c^2$`
- Display: `$$ ... $$`, usually on its own lines:

  ```markdown
  $$
  \int_0^1 x^2\,dx = \frac{1}{3}
  $$
  ```

Escape a literal dollar sign as `\$`. An opening `$` followed by a space, or a closing `$` followed by a digit, is
treated as plain text, so prices such as "$5 and $10" are left alone. With the feature off, dollar signs are ordinary
text.

### Links between documents and headings

Relative links to other documents in the bundle become internal PDF links. The target must be listed in `documents`.

```markdown
[the goals](intro.md#goals)      <!-- a heading in another document -->
[the details](details.md)        <!-- the first heading of a document -->
[below](#goals)                  <!-- a heading in the same document -->
```

- Fragments match the heading's text slug (`## Goals` is `goals`). If two headings in a document share a slug, the link
  is rejected as ambiguous; give one a custom id, as in `## Goals {#goals-2}`, or use the full generated id
  (`<document>--<slug>`, such as `intro--goals`).
- Links with a URL scheme (`https://…`, `mailto:…`) are left as they are.
- A link that cannot be resolved fails the render and is reported with its file and line.

### Cross-reference text

A link with empty text, or with the title `"xref"`, gets generated text: `Section 2.1, p. 4`.

```markdown
See [](intro.md#goals) for the goals, or [anything](intro.md#goals "xref").
```

The section part is omitted when `numbering.enabled` is false or the heading is deeper than `numbering.depth`; the page
part is omitted when `page.numbers` is false. If both are omitted, the heading text is used.

### Images

Use ordinary Markdown images with paths relative to the document that contains them:

```markdown
![Floor plan](images/floor-plan.png)
```

Images are inlined into the PDF, so the output is self-contained. PNG, JPEG, GIF, WebP, and SVG are supported. Remote
(`http://`, `https://`) images are not; add the file to the bundle. Paths must stay inside the bundle.

## Repository layout

| Path        | Contents                                                                                  |
| ----------- | ----------------------------------------------------------------------------------------- |
| `cmd/`      | Entry points. `cmd/spun` is the `spun` command.                                           |
| `internal/` | The implementation: bundle loading, Markdown rendering, templates, features, and PDF output. |
| `docs/`     | Home for the business's plans and operating documents, as `spun` bundles.             |
