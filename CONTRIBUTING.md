# Contributing

Thanks for helping with Spun Records and Lounge. This repository holds both the business's documents and `spun`, the
tool that renders them. See the [README](README.md) for what `spun` does and the full `spun.yaml` reference.

## Development setup

- **Go**: the version declared in [`go.mod`](go.mod).
- **Chromium**: `spun` prints PDFs through a headless Chromium or Chrome, and the render tests need one. Point
  `SPUN_CHROMIUM` at the executable, or put `chromium`, `chromium-browser`, `google-chrome`, or `google-chrome-stable` on
  your `PATH`:

  ```sh
  export SPUN_CHROMIUM=/path/to/chromium
  ```

  If Chromium cannot start its sandbox (in a container, for example), also set `SPUN_CHROMIUM_NO_SANDBOX=1`, as CI does.

## Build, vet, and test

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs these three commands. Run them before opening a pull
request:

```sh
go build ./...
go vet ./...
go test ./...
```

## Adding a feature

Optional authoring features (such as math) live in the registry in [`internal/feature`](internal/feature). A feature is
off until a bundle enables it. [`internal/features/math`](internal/features/math) is the worked example.

1. **Create a package** at `internal/features/<name>/`.
2. **Implement `feature.Feature`**, which needs only `Name()`. The name is the key used under `features:` in
   `spun.yaml`:

   ```go
   const Name = "math"

   type Feature struct{}

   func (Feature) Name() string { return Name }
   ```

3. **Implement the capability interfaces the feature needs.** Each is optional:
   - `feature.MarkdownExtender`: `Extenders()` returns goldmark extensions. Math uses this to add its inline and block
     parsers and renderer.
   - `feature.Stage`: `StageName()`, `Phase()` (`feature.PreRender` or `feature.PostRender`), and
     `Process(doc string) (string, error)` to transform the source or the HTML.
   - `feature.CommandProvider`: `Commands()` returns extra cobra commands.
   - `feature.Assets`: `HeadHTML()`, `Styles()`, and `Scripts()` add to the page head. Math uses this to embed KaTeX.
4. **Register from `init()`** on the shared registry. Registering a name twice panics:

   ```go
   func init() {
       feature.Default.Register(Feature{})
   }
   ```

5. **Blank-import the package in [`cmd/spun/main.go`](cmd/spun/main.go)** so the `init()` runs in the binary:

   ```go
   _ "github.com/bengearig/spun-records-and-lounge/internal/features/math"
   ```

6. **Enable it in `spun.yaml`**. Unknown feature names are rejected when the bundle loads:

   ```yaml
   features:
     math: true
   ```

7. **Test it.** Follow `math_test.go` and `render_test.go` in the example package: look the feature up with
   `feature.Default.Lookup`, and cover parsing and rendering. Document the feature in the README's authoring section.

## Adding a bundle

Bundles live under `docs/`, one directory per bundle: `docs/<name>/`. Each contains:

```text
docs/<name>/
├── spun.yaml        # title, documents, and any options
├── intro.md         # Markdown files, listed in spun.yaml
└── images/
    └── photo.png    # images referenced from the Markdown, relative to the file
```

1. Create `docs/<name>/spun.yaml` with at least `title` and `documents`; the README documents every key.
2. Add the Markdown files and list them under `documents` in reading order. Link between them with relative links such
   as `[goals](other.md#goals)`.
3. Put images inside the bundle and reference them with relative paths.
4. Render it and check the PDF:

   ```sh
   go run ./cmd/spun docs render docs/<name> -o /tmp/<name>.pdf
   ```

   A broken link or missing image fails the render with the file and line.

## RALPH workflow

Planning follows the RALPH method. Plans (initiatives, documents, milestones, and tasks) live in ralph's store, reached
through the `ralph` MCP server, not in ad-hoc files in the repository. [`CLAUDE.md`](CLAUDE.md) describes the workflow
and its slash commands. Branches named `ralph/M-<n>-...` correspond to milestones.

## Pull requests

- CI must pass: build, vet, and test, as above.
- One milestone per pull request. Keep a pull request to the tasks of its milestone, and open follow-up work as new
  tasks instead of widening the change.
- Include or update tests for behavior you change, and update the README when user-facing behavior changes.
