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

The available features are `math` and `costs`:

```yaml
features:
  math: true          # shorthand
  # or
  math:
    enabled: true
  costs:
    file: data/startup-costs.yaml   # relative to the bundle
```

`costs` requires `file`, the path of its data file. The path is relative to the bundle and must stay inside it. See
[Costs](#costs).

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

### Costs

With `features: {costs: {file: <path>}}`, figures and tables come from one YAML data file, so numbers in the prose are
never typed by hand. The directives below insert them.

#### Data file

Unknown fields are errors. Every `id` is a lowercase slug (letters, digits, hyphens).

```yaml
levels:                     # ordered; the first is the base; at least one
  - id: base                # slug
    name: "Base: 1,500 sq ft"
    description: ""         # optional free text
    sqft: 1500              # optional positive whole number, documentation only, never used in arithmetic
    reserve_months:         # months of reserve this level ADDS; optional, default 0/0/0
      low: 3
      expected: 3
      high: 6
categories:                 # ordered; order drives table order; at least one
  - id: space               # slug
    name: Space
items:                      # one-time startup items; at least one
  - id: security-deposit    # slug, unique across items and monthly
    category: space         # must name a category
    level: base             # must name a level
    name: Security deposit
    low: 3000               # amounts: non-negative, at most 2 decimals, plain digits (no $ or commas)
    expected: 6000
    high: 12000
    status: estimate        # estimate | quote | paid
    owner_input: true       # optional, default false: the owner still has to supply the real amount
    source: "D-25: Space, lease security deposit"   # optional free text
    notes: ""               # optional free text
monthly:                    # monthly operating items, used only to size the reserve; may be empty or absent
  - id: rent                # same fields as an item but no `category`; `level` is the level from which the cost applies; amounts are per month
    level: base
    name: Rent and NNN
    low: 2000
    expected: 3000
    high: 4500
    status: estimate
```

| Field                | Meaning                                                                                              |
| -------------------- | ---------------------------------------------------------------------------------------------------- |
| `level`              | Required on every item and monthly item. An item belongs to exactly one level.                       |
| `status`             | `estimate`, `quote`, or `paid`.                                                                      |
| `owner_input`        | `true` when the owner still has to supply the real amount. Defaults to `false`.                      |
| `reserve_months`     | On a level: the months of reserve that level adds, as `low`, `expected`, and `high` whole months.    |
| `low`, `expected`, `high` | The range for an item, with `low <= expected <= high`. For a monthly item, the range is per month. |

#### Money

- Amounts are plain digits with at most two decimals: `3000`, `1000.50`. Do not write `$` or thousands separators.
- Amounts are never negative and are kept as exact cents, not floating point.
- Figures display as `$1,234`, or `$1,234.50` when there are cents.
- A sum that overflows is an error.

#### Funding levels

Levels stage the plan like stretch goals: the first level is the base, and later levels are ordered stretch goals. Each
level has an `Adds` figure and a cumulative `Goal`:

| Figure   | Meaning                                                                                           |
| -------- | ------------------------------------------------------------------------------------------------- |
| `Adds`   | What the level costs on top of the one before it. For the base, `Adds` equals `Goal`.             |
| `Goal`   | The cumulative amount to raise to reach the level: all items through the level, plus its reserve. |

The reserve at a level is the running total of reserve months through that level, times the running monthly total of
every monthly item at that level or earlier. Because amounts and months are never negative, a goal never decreases from
one level to the next.

#### Totals

| Path              | Figure                                                          |
| ----------------- | --------------------------------------------------------------- |
| `startup`         | All one-time items, across every level.                         |
| `monthly`         | Monthly operating cost at the last level.                       |
| `reserve`         | Reserve at the last level.                                      |
| `total`           | `startup` plus `reserve`: the last level's goal.                |
| `category.<id>`   | One category's items across every level.                        |
| `item.<id>`       | One item or monthly item.                                       |
| `level.<id>`      | What that level adds.                                           |
| `goal.<id>`       | The cumulative goal to reach that level.                        |

#### Inline figures

Write `{{costs:<path>}}` in prose, using a path from the table above:

```markdown
We are raising {{costs:total}}. The base needs {{costs:goal.base.expected}}.
```

- Add `.low`, `.expected`, or `.high` to get one figure, as in `{{costs:total.high}}`.
- With no suffix the output is a range, `$low to $high`, or just `$value` when low and high are equal.
- Put a backslash before the first brace, `\{{costs:total}}`, to keep the text literal.
- Directives inside code spans and code blocks are left alone.

#### Tables

A fenced code block with the info string `costs` becomes a table. The body is one line: the kind, then optional
`key=value` options.

````markdown
```costs
items category=space level=base source=true
```
````

| Kind         | Columns                                       | Rows                                                                                                   |
| ------------ | --------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `categories` | Category, Low, Expected, High                 | One per category, then `Startup subtotal`, `Cash reserve`, and `Total to raise`.                       |
| `items`      | Item, Low, Expected, High, Status             | Items in category order. Options: `category=<id>`, `level=<id>`, and `source=true` to add a Source column. |
| `reserve`    | Item, Level, Low, Expected, High              | One per monthly item (per month), then `Monthly total`, `Reserve months`, and `Reserve`.               |
| `goals`      | Level, Low, Expected, High                    | One per level, showing its cumulative goal.                                                            |
| `adds`       | Level, Low, Expected, High                    | One per level, showing what it adds.                                                                   |
| `level`      | Item, Low, Expected, High, Status             | The items of one level, then `Items subtotal`, `Added reserve`, `Adds`, and `Goal`. Requires `id=<level id>`. |

Text from the data file is HTML-escaped, and money cells are right aligned.

#### Errors

A problem in the data file stops the render and names the file path from `spun.yaml` and the item or level id, or the
list and index when the id is missing. Examples:

- unknown fields, a missing `file`, a `file` that is absolute, outside the bundle, or missing;
- no levels, categories, or items; a non-slug or duplicate id (items and monthly items share one namespace);
- an item with a missing or unknown `category` or `level`, or a monthly item with a `category`;
- a missing `name`, an invalid `status`, `low <= expected <= high` violated, a bad money value;
- a level `sqft` that is not positive, or `reserve_months` that are negative or out of order.

A bad directive stops the render with the document line and the directive text, for an unknown path, id, or suffix. A
bad `costs` block reports its line for an unknown kind or option, an unknown id, or a missing `id` on a `level` table.

#### Checking the totals

```sh
spun costs totals <bundle-dir>
```

Prints each category, the startup subtotal, monthly operating cost, reserve, and total to raise, then what each level
adds and its goal. It requires `costs` to be enabled in the bundle and exits non-zero with the error if the data file is
invalid.

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

## License

| Paths                                           | License                                  | File                               |
| ----------------------------------------------- | ---------------------------------------- | ---------------------------------- |
| `cmd/`, `internal/`, and other source code      | MIT                                      | [`LICENSE`](LICENSE)               |
| `docs/` (business plans, reports, written content) | Creative Commons Attribution 4.0 International | [`LICENSE-CONTENT`](LICENSE-CONTENT) |

Third-party assets keep their own licenses. In particular, the vendored KaTeX files under
`internal/features/math/assets/` are covered by the license in that directory, not by `LICENSE`.
