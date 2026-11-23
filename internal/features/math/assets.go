package math

import (
	"embed"
	"encoding/base64"
	"regexp"
	"strings"
	"sync"
)

//go:embed assets
var assetsFS embed.FS

const assetsDir = "assets/"

// ReadyFlag and ErrorFlag are the window properties set by the render script.
const (
	ReadyFlag = "__spunReady"
	ErrorFlag = "__spunError"
)

const renderScript = `
(function () {
  function run() {
    var failure = "";
    document.querySelectorAll(".math").forEach(function (el) {
      var tex = el.dataset.tex;
      try {
        katex.render(tex, el, {
          displayMode: el.classList.contains("display"),
          throwOnError: true
        });
      } catch (e) {
        if (!failure) failure = e.message + " (TeX: " + tex + ")";
      }
    });
    if (failure) window.__spunError = failure;
    window.__spunReady = true;
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", run);
  } else {
    run();
  }
})();
`

var fontRule = regexp.MustCompile(`url\(fonts/([A-Za-z0-9_-]+)\.woff2\) format\("woff2"\),url\([^)]*\) format\("woff"\),url\([^)]*\) format\("truetype"\)`)

var (
	cssOnce sync.Once
	cssText string
)

func mustRead(name string) string {
	b, err := assetsFS.ReadFile(assetsDir + name)
	if err != nil {
		panic("math: missing embedded asset " + name + ": " + err.Error())
	}
	return string(b)
}

// inlineCSS returns the KaTeX stylesheet with each font embedded as a base64
// woff2 data URI, so printing needs no network or file access.
func inlineCSS() string {
	cssOnce.Do(func() {
		cssText = fontRule.ReplaceAllStringFunc(mustRead("katex.min.css"), func(m string) string {
			name := fontRule.FindStringSubmatch(m)[1]
			enc := base64.StdEncoding.EncodeToString([]byte(mustRead("fonts/" + name + ".woff2")))
			return `url(data:font/woff2;base64,` + enc + `) format("woff2")`
		})
	})
	return cssText
}

// HeadHTML implements feature.Assets.
func (Feature) HeadHTML() string { return "" }

// Styles implements feature.Assets.
func (Feature) Styles() string { return inlineCSS() }

// Scripts implements feature.Assets; the script defers to DOMContentLoaded
// because it runs from the head.
func (Feature) Scripts() string {
	return strings.TrimSpace(mustRead("katex.min.js")) + "\n" + strings.TrimSpace(renderScript)
}
