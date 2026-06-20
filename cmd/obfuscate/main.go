// obfuscate – deterministic class/id/data-attribute obfuscator for static sites.
//
// Walks a directory of built HTML/CSS/JS files, extracts all class names, IDs,
// and data attribute names, generates a deterministic replacement token for each,
// and rewrites all files using the obfuscated tokens.
//
// Protected names (never obfuscated): "dark", "not-prose", data-contact-form,
// data-pgp-block, data-newsletter-checkbox-id.
//
// Usage:
//
//	go run ./cmd/obfuscate [flags] <public-dir>
//
// Flags:
//
//	--verbose        Print detailed progress
//	--assets         Also rename asset files (CSS/JS) using mapping
//	--dry-run        Print what would change without writing
//	--output-map     Write mapping JSON to this path (default: public/obfuscate-map.json)
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// Protected names that must never be obfuscated.
var protectedSet = map[string]bool{
	// Theme-critical classes that must not be obfuscated
	"class:dark":              true,
	"class:not-prose":         true,
	"class:sr-only":           true,
	"class:not-sr-only":       true,
	"class:focus:not-sr-only": true,
	"class:skip-to-content":   true,
	// Data attributes used by contact form and other interactive elements
	"data:contact-form":           true,
	"data:pgp-block":              true,
	"data:newsletter-checkbox-id": true,
	"data:contact-submit":         true,
	"data:contact-loading":        true,
	"data:contact-toast":          true,
	"data:first-name":             true,
	"data:last-name":              true,
	"data:email":                  true,
	"data:subject":                true,
	"data:text":                   true,
	"data:encrypt":                true,
	"data:hp":                     true,
	"data:pgp-error":              true,
	"data:umami-event":            true,
	// Search JS data attributes (must not be obfuscated)
	"data:index-url":              true,
	"data:base-path":              true,
	"data:visible-count":          true,
	"id:theme-toggle":             true,
	"id:mobile-menu-button":       true,
	"id:mobile-drawer":            true,
	"id:mobile-drawer-overlay":    true,
	"id:mobile-drawer-close":      true,
	"id:contact-config":           true,
	// Blog search IDs (must not be obfuscated — JS references them)
	"id:blog-search":              true,
	"id:blog-search-input":        true,
	"id:blog-search-tags":         true,
	"id:blog-search-status":       true,
	"id:blog-server-list":         true,
	"id:blog-results":             true,
	"id:blog-clear-filters":       true,
	"id:blog-toggle-tags":         true,
	"id:reading-server-list":      true,
}

// nameItem represents a discovered name to potentially obfuscate.
type nameItem struct {
	Type string // "class", "id", or "data"
	Name string
}

func (n nameItem) Key() string {
	return n.Type + ":" + n.Name
}

func main() {
	verbose := flag.Bool("verbose", false, "Print detailed progress")
	assets := flag.Bool("assets", false, "Also rename asset files using mapping")
	dryRun := flag.Bool("dry-run", false, "Print what would change without writing")
	outputMap := flag.String("output-map", "", "Write mapping JSON to this path")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: obfuscate [flags] <public-dir>")
		os.Exit(1)
	}
	publicDir := flag.Arg(0)
	mapPath := *outputMap
	if mapPath == "" {
		mapPath = filepath.Join(publicDir, "obfuscate-map.json")
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Scanning %s ...\n", publicDir)
	}

	// Collect all files
	files := walkFiles(publicDir)

	// Phase 1: extract all names
	allNames := collectNames(files, *verbose)

	// Phase 2: build mapping
	mapping := buildMapping(allNames)

	if *verbose {
		fmt.Fprintf(os.Stderr, "Found %d unique names, mapping %d tokens\n", len(allNames), len(mapping))
	}

	// Phase 3: rewrite files
	var mu sync.Mutex
	var totalReplacements int
	var wg sync.WaitGroup

	// Process files in parallel batches (3× CPU cores)
		workers := runtime.NumCPU() * 3
		if *verbose {
			fmt.Fprintf(os.Stderr, "Using %d workers (%d CPUs × 3)\n", workers, runtime.NumCPU())
		}
		sem := make(chan struct{}, workers)
		for _, f := range files {
			wg.Add(1)
			sem <- struct{}{}
			go func(filePath string) {
				defer wg.Done()
				defer func() { <-sem }()

				count := replaceInFile(filePath, mapping, *dryRun, *verbose)
				mu.Lock()
			totalReplacements += count
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	if *verbose {
		fmt.Fprintf(os.Stderr, "Total replacements: %d across %d files\n", totalReplacements, len(files))
	}

	// Write mapping file if not dry run
	if !*dryRun {
		mapData, _ := json.MarshalIndent(mapping, "", "  ")
		os.WriteFile(mapPath, mapData, 0644)
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote mapping to %s\n", mapPath)
		}
	}

	fmt.Printf("Obfuscation complete. %d files processed, %d replacements.\n", len(files), totalReplacements)

	// Suppress unused
	_ = assets
}

func walkFiles(dir string) []string {
	var files []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".html", ".htm", ".css", ".js":
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func collectNames(files []string, verbose bool) []nameItem {
	workers := runtime.NumCPU() * 3
	var (
		mu  sync.Mutex
		all = make(map[string]nameItem)
		wg  sync.WaitGroup
		sem = make(chan struct{}, workers)
	)

	for _, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-sem }()
			names := extractNames(filePath)
			mu.Lock()
			for _, n := range names {
				if !protectedSet[n.Key()] {
					all[n.Key()] = n
				}
			}
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	result := make([]nameItem, 0, len(all))
	for _, v := range all {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key() < result[j].Key()
	})
	return result
}

func extractNames(filePath string) []nameItem {
	ext := strings.ToLower(filepath.Ext(filePath))
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(content)

	switch ext {
	case ".html", ".htm":
		return extractHTMLNames(s)
	case ".css":
		return extractCSSNames(s)
	case ".js":
		return extractJSNames(s)
	}
	return nil
}

// ─── HTML extraction ──────────────────────────────────────────────────────────

func extractHTMLNames(content string) []nameItem {
	seen := make(map[string]bool)
	var names []nameItem

	// Parse HTML and walk nodes for class/id/data-* attributes
	doc, err := html.Parse(strings.NewReader(content))
	if err == nil {
		extractFromNode(doc, seen, &names)
	}

	// Also extract from inline <style> blocks
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
	for _, m := range styleRe.FindAllStringSubmatch(content, -1) {
		for _, n := range extractCSSNames(m[1]) {
			if !seen[n.Key()] {
				seen[n.Key()] = true
				names = append(names, n)
			}
		}
	}

	// And inline <script> blocks (skip JSON-LD and import maps)
	scriptRe := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	for _, m := range scriptRe.FindAllStringSubmatch(content, -1) {
		attrs := m[1]
		if strings.Contains(attrs, "src=") {
			continue
		}
		if strings.Contains(attrs, "application/ld+json") || strings.Contains(attrs, "importmap") {
			continue
		}
		for _, n := range extractJSNames(m[2]) {
			if !seen[n.Key()] {
				seen[n.Key()] = true
				names = append(names, n)
			}
		}
	}

	return names
}

func extractFromNode(n *html.Node, seen map[string]bool, names *[]nameItem) {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			val := attr.Val
			switch key {
			case "class":
				for _, cls := range strings.Fields(val) {
					cls = strings.Trim(cls, `"'`)
					if cls == "" {
						continue
					}
					it := nameItem{Type: "class", Name: cls}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						*names = append(*names, it)
					}
				}
			case "id":
				if val != "" {
					it := nameItem{Type: "id", Name: val}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						*names = append(*names, it)
					}
				}
			default:
				if strings.HasPrefix(key, "data-") {
					dataName := key[5:]
					it := nameItem{Type: "data", Name: dataName}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						*names = append(*names, it)
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractFromNode(c, seen, names)
	}
}

// ─── CSS extraction ───────────────────────────────────────────────────────────

func extractCSSNames(content string) []nameItem {
	seen := make(map[string]bool)
	var names []nameItem

	// Extract from selector blocks (before {)
	blockRe := regexp.MustCompile(`([^{}]+)\{`)
	for _, m := range blockRe.FindAllStringSubmatch(content, -1) {
		selector := m[1]
		// class selectors: .foo or .md\:flex or .min-h-\[calc(...)\] (with escape chars)
		for _, cm := range regexp.MustCompile(`\.([A-Za-z0-9_\\:\/\-\[\]\(\)\.%]+)`).FindAllStringSubmatch(selector, -1) {
			// Unescape CSS class names (e.g. md\:flex -> md:flex, \[ -> [)
			name := strings.ReplaceAll(cm[1], "\\:", ":")
			name = strings.ReplaceAll(name, "\\/", "/")
			name = strings.ReplaceAll(name, "\\[", "[")
			name = strings.ReplaceAll(name, "\\]", "]")
			name = strings.ReplaceAll(name, "\\(", "(")
			name = strings.ReplaceAll(name, "\\)", ")")
			name = strings.ReplaceAll(name, "\\.", ".")
			name = strings.ReplaceAll(name, "\\%", "%")
			it := nameItem{Type: "class", Name: name}
			if !seen[it.Key()] {
				seen[it.Key()] = true
				names = append(names, it)
			}
		}
		// id selectors: #foo
		for _, im := range regexp.MustCompile(`#([A-Za-z0-9_-]+)`).FindAllStringSubmatch(selector, -1) {
			it := nameItem{Type: "id", Name: im[1]}
			if !seen[it.Key()] {
				seen[it.Key()] = true
				names = append(names, it)
			}
		}
		// data-* attribute selectors: [data-foo]
		for _, am := range regexp.MustCompile(`\[\s*data-([A-Za-z0-9_-]+)`).FindAllStringSubmatch(selector, -1) {
			it := nameItem{Type: "data", Name: am[1]}
			if !seen[it.Key()] {
				seen[it.Key()] = true
				names = append(names, it)
			}
		}
	}

	return names
}

// ─── JS extraction ────────────────────────────────────────────────────────────

func extractJSNames(content string) []nameItem {
	seen := make(map[string]bool)
	var names []nameItem

	patterns := []struct {
		re       *regexp.Regexp
		typ      string
		submatch int
	}{
		// getElementById('foo')
		{regexp.MustCompile(`getElementById\(\s*['"]([A-Za-z0-9_-]+)['"]\s*\)`), "id", 1},
		// getElementsByClassName('a b')
		{regexp.MustCompile(`getElementsByClassName\(\s*['"]([A-Za-z0-9_ -]+)['"]\s*\)`), "class", 1},
		// querySelector('.foo') or querySelector('#bar')
		{regexp.MustCompile(`querySelector(?:All)?\s*\(\s*['"]([^'"]+)['"]\s*\)`), "selector", 1},
		// getAttribute('data-foo') and setAttribute('data-foo', ...)
		{regexp.MustCompile(`[gs]etAttribute\(\s*['"]data-([A-Za-z0-9_-]+)['"]\s*[,)]`), "data", 1},
		// dataset.prop
		{regexp.MustCompile(`\.dataset\.([A-Za-z0-9_$]+)`), "data-camel", 1},
		// className = "class1 class2" or className='class1 class2'
		{regexp.MustCompile(`\.className\s*=\s*['"]([^'"]+)['"]`), "class", 1},
		// class="..." inside template literals / innerHTML strings
		{regexp.MustCompile(`class="([^"]*)"`), "class", 1},
		// class='...' inside template literals (less common but handle it)
		{regexp.MustCompile(`class='([^']*)'`), "class", 1},
	}

	for _, p := range patterns {
		for _, m := range p.re.FindAllStringSubmatch(content, -1) {
			val := m[p.submatch]
			switch p.typ {
			case "class":
				for _, cls := range strings.Fields(val) {
					if cls == "" {
						continue
					}
					it := nameItem{Type: "class", Name: cls}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						names = append(names, it)
					}
				}
			case "id":
				it := nameItem{Type: "id", Name: val}
				if !seen[it.Key()] {
					seen[it.Key()] = true
					names = append(names, it)
				}
			case "data":
				it := nameItem{Type: "data", Name: val}
				if !seen[it.Key()] {
					seen[it.Key()] = true
					names = append(names, it)
				}
			case "data-camel":
				// Convert camelCase to kebab-case
				kebab := strings.ToLower(regexp.MustCompile(`([A-Z])`).ReplaceAllString(val, "-$1"))
				it := nameItem{Type: "data", Name: kebab}
				if !seen[it.Key()] {
					seen[it.Key()] = true
					names = append(names, it)
				}
			case "selector":
				// Parse classes, ids, data attrs from selector string
				for _, cm := range regexp.MustCompile(`\.([A-Za-z0-9_-]+)`).FindAllStringSubmatch(val, -1) {
					it := nameItem{Type: "class", Name: cm[1]}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						names = append(names, it)
					}
				}
				for _, im := range regexp.MustCompile(`#([A-Za-z0-9_-]+)`).FindAllStringSubmatch(val, -1) {
					it := nameItem{Type: "id", Name: im[1]}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						names = append(names, it)
					}
				}
				for _, am := range regexp.MustCompile(`\[\s*data-([A-Za-z0-9_-]+)`).FindAllStringSubmatch(val, -1) {
					it := nameItem{Type: "data", Name: am[1]}
					if !seen[it.Key()] {
						seen[it.Key()] = true
						names = append(names, it)
					}
				}
			}
		}
	}

	return names
}

// ─── Mapping ──────────────────────────────────────────────────────────────────

func buildMapping(names []nameItem) map[string]string {
	mapping := make(map[string]string)
	for _, it := range names {
		if protectedSet[it.Key()] {
			continue
		}
		prefix := "c"
		switch it.Type {
		case "id":
			prefix = "i"
		case "data":
			prefix = "d"
		}
		mapping[it.Key()] = deterministicToken(it.Key(), prefix)
	}
	return mapping
}

func deterministicToken(key, prefix string) string {
	hash := sha256.Sum256([]byte(key))
	hexStr := hex.EncodeToString(hash[:])
	// Use seed from first 8 hex chars of hash for length jitter
	seed := int(hash[0])<<24 | int(hash[1])<<16 | int(hash[2])<<8 | int(hash[3])
	baseLen := len(key)*3/5 + 6
	if baseLen < 8 {
		baseLen = 8
	}
	if baseLen > 32 {
		baseLen = 32
	}
	// Deterministic jitter based on seed
	jitter := (seed%12 - 6)
	n := baseLen + jitter
	if n < 8 {
		n = 8
	}
	if n > 32 {
		n = 32
	}
	if n > len(hexStr) {
		n = len(hexStr)
	}
	return prefix + hexStr[:n-len(prefix)]
}

// ─── File rewriting ───────────────────────────────────────────────────────────

func replaceInFile(filePath string, mapping map[string]string, dryRun, verbose bool) int {
	ext := strings.ToLower(filepath.Ext(filePath))
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	original := string(content)
	var result string

	switch ext {
	case ".html", ".htm":
		result = replaceHTML(original, mapping)
	case ".css":
		result = replaceCSS(original, mapping)
	case ".js":
		result = replaceJS(original, mapping)
	default:
		return 0
	}

	if result == original {
		return 0
	}

	// Count actual changes by comparing original vs replaced classes
	count := 0
	// Count based on total bytes changed
	if len(result) != len(original) {
		count = 1 // non-trivial change
	}
	// More precise: count how many mapping tokens appear in the result
	for _, token := range mapping {
		if strings.Count(result, token) > 0 && strings.Count(original, token) == 0 {
			count++
		}
	}

	if dryRun {
		if verbose {
			fmt.Fprintf(os.Stderr, "[dry-run] %s: %d tokens changed\n", filePath, count)
		}
		return count
	}

	if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", filePath, err)
		return 0
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Rewrote %s (%d changes)\n", filePath, count)
	}

	return count
}

func replaceHTML(content string, mapping map[string]string) string {
	// First, handle inline scripts and styles (before HTML attribute replacement)
	result := replaceJSText(content, mapping)
	result = replaceCSSText(result, mapping)

	// Replace class attributes: class="a b c" or class=cls
	classRe := regexp.MustCompile(`class="([^"]*)"|class=([^\s>]+)`)
	result = classRe.ReplaceAllStringFunc(result, func(match string) string {
		parts := classRe.FindStringSubmatch(match)
		val := parts[1]
		if val == "" {
			val = parts[2]
		}
		if val == "" {
			return match
		}
		var newClasses []string
		for _, cls := range strings.Fields(val) {
			cls = strings.Trim(cls, `"'`)
			if cls == "" {
				continue
			}
			if token := mapping["class:"+cls]; token != "" {
				newClasses = append(newClasses, token)
			} else {
				newClasses = append(newClasses, cls)
			}
		}
		return `class="` + strings.Join(newClasses, " ") + `"`
	})

	// Replace id attributes: id="foo" or id=bar (minified HTML may remove quotes)
		idRe := regexp.MustCompile(`\bid="([^"]*)"|\bid=([^\s>]+)`)
		result = idRe.ReplaceAllStringFunc(result, func(match string) string {
			parts := idRe.FindStringSubmatch(match)
			val := parts[1]
			if val == "" {
				val = parts[2]
			}
			if val == "" {
				return match
			}
			if token := mapping["id:"+val]; token != "" {
				return `id="` + token + `"`
			}
			return match
		})

	// Replace data-* attribute names
	dataRe := regexp.MustCompile(`\b(data-[A-Za-z0-9_-]+)(\s*=)`)
	result = dataRe.ReplaceAllStringFunc(result, func(match string) string {
		parts := dataRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		dataName := parts[1][5:] // strip "data-" prefix
		if token := mapping["data:"+dataName]; token != "" {
			return `data-` + token + parts[2]
		}
		return match
	})

	// Replace aria-* and other ID-reference attributes
	for _, attr := range []string{"aria-controls", "aria-labelledby", "aria-describedby", "aria-owns", "aria-activedescendant", "for", "list"} {
		attrRe := regexp.MustCompile(`\b` + attr + `="([^"]*)"`)
		result = attrRe.ReplaceAllStringFunc(result, func(match string) string {
			parts := attrRe.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			var refs []string
			for _, ref := range strings.Fields(parts[1]) {
				if token := mapping["id:"+ref]; token != "" {
					refs = append(refs, token)
				} else {
					refs = append(refs, ref)
				}
			}
			return attr + `="` + strings.Join(refs, " ") + `"`
		})
	}

	// Replace href="#id" fragment references
	hrefRe := regexp.MustCompile(`\bhref="#([^"]*)"`)
	result = hrefRe.ReplaceAllStringFunc(result, func(match string) string {
		parts := hrefRe.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if token := mapping["id:"+parts[1]]; token != "" {
			return `href="#` + token + `"`
		}
		return match
	})

	return result
}

func replaceHTMLNode(n *html.Node, mapping map[string]string) {
	if n.Type == html.ElementNode {
		for i, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			switch key {
			case "class":
				var newClasses []string
				for _, cls := range strings.Fields(attr.Val) {
					token := mapping["class:"+cls]
					if token != "" {
						newClasses = append(newClasses, token)
					} else {
						newClasses = append(newClasses, cls)
					}
				}
				n.Attr[i].Val = strings.Join(newClasses, " ")
			case "id":
				token := mapping["id:"+attr.Val]
				if token != "" {
					n.Attr[i].Val = token
				}
			case "for", "list", "aria-controls", "aria-labelledby", "aria-describedby", "aria-owns", "aria-activedescendant":
				var refs []string
				for _, ref := range strings.Fields(attr.Val) {
					token := mapping["id:"+ref]
					if token != "" {
						refs = append(refs, token)
					} else {
						refs = append(refs, ref)
					}
				}
				n.Attr[i].Val = strings.Join(refs, " ")
			case "href":
				// Replace #id fragment references
				if strings.HasPrefix(attr.Val, "#") {
					token := mapping["id:"+attr.Val[1:]]
					if token != "" {
						n.Attr[i].Val = "#" + token
					}
				}
			default:
				if strings.HasPrefix(key, "data-") {
					dataName := key[5:]
					token := mapping["data:"+dataName]
					if token != "" {
						n.Attr[i].Key = "data-" + token
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		replaceHTMLNode(c, mapping)
	}
}

func replaceCSS(content string, mapping map[string]string) string {
	// Replace class selectors: .original -> .token
	// Sort by length descending so longer names get replaced first, preventing
	// partial matches (e.g. "flex" matching inside "flex-col")
	sorted := sortMappingByLength(mapping)
	for _, key := range sorted {
		if !strings.HasPrefix(key, "class:") {
			continue
		}
		orig := key[6:]
		// Escape special chars to match the CSS-escaped form in selectors
		var cssForm strings.Builder
		for _, r := range orig {
			switch r {
			case ':', '[', ']', '(', ')', '.', '%', '/':
				cssForm.WriteByte('\\')
				cssForm.WriteRune(r)
			default:
				cssForm.WriteRune(r)
			}
		}
		// Use \Q...\E to match the CSS form literally, then non-{ chars until {
		re := regexp.MustCompile(`\.\Q` + cssForm.String() + `\E([^{]*)\{`)
		content = re.ReplaceAllString(content, "."+mapping[key]+"$1{")
	}
	// Replace id selectors
	for _, key := range sorted {
		if !strings.HasPrefix(key, "id:") {
			continue
		}
		orig := key[3:]
		re := regexp.MustCompile(`#` + regexp.QuoteMeta(orig) + `\b`)
		content = re.ReplaceAllString(content, "#"+mapping[key])
	}
	// Replace data-* selectors
	for _, key := range sorted {
		if !strings.HasPrefix(key, "data:") {
			continue
		}
		orig := key[5:]
		re := regexp.MustCompile(`\[\s*data-` + regexp.QuoteMeta(orig) + `\b`)
		content = re.ReplaceAllString(content, "[data-"+mapping[key])
	}
	return content
}

// cssEscape returns the CSS-selector-escaped form of a class name.
// e.g. "md:flex" -> "md\:flex", "min-h-[calc(100vh-10rem)]" -> "min-h-\[calc\(100vh-10rem\)\]"
func cssEscape(name string) string {
	r := strings.NewReplacer(
		":", "\\:",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		".", "\\.",
		"%", "\\%",
		"/", "\\/",
	)
	return r.Replace(name)
}

func sortMappingByLength(mapping map[string]string) []string {
	keys := make([]string, 0, len(mapping))
	for k := range mapping {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	return keys
}

func replaceJS(content string, mapping map[string]string) string {
	// Replace class/id/data names inside string arguments in DOM method calls.
	sorted := sortMappingByLength(mapping)
	for _, key := range sorted {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		typ, orig := parts[0], parts[1]
		token := mapping[key]

		switch typ {
		case "class":
			for _, method := range []string{"classList.add", "classList.remove", "classList.toggle", "classList.contains", "classList.replace", "getElementsByClassName"} {
				content = replaceStringArg(content, method, orig, token)
			}
			content = replaceInSelectorStrings(content, orig, token)
			// Replace class names inside class="..." strings (innerHTML templates)
			content = replaceInHTMLClassStrings(content, orig, token)
			// Replace class names inside className="..." assignments
			content = replaceInClassNameAssignments(content, orig, token)
		case "id":
			content = replaceStringArg(content, "getElementById", orig, token)
			content = replaceInSelectorStrings(content, "#"+orig, "#"+token)
		case "data":
			re := regexp.MustCompile(`([gs]etAttribute\(\s*['"]data-)` + regexp.QuoteMeta(orig) + `(['"]\s*[,)])`)
			content = re.ReplaceAllString(content, "${1}"+token+"${2}")
		}
	}
	return content
}

// replaceStringArg replaces a quoted string argument in a method call.
func replaceStringArg(content, method, orig, token string) string {
	prefix := method + "("
	result := content
	offset := 0
	for {
		idx := strings.Index(result[offset:], prefix)
		if idx == -1 {
			break
		}
		idx += offset
		end := strings.Index(result[idx:], ")")
		if end == -1 {
			break
		}
		end += idx
		args := result[idx+len(prefix) : end]
		newArgs := strings.ReplaceAll(args, "\""+orig+"\"", "\""+token+"\"")
		newArgs = strings.ReplaceAll(newArgs, "'"+orig+"'", "'"+token+"'")
		if newArgs != args {
			result = result[:idx+len(prefix)] + newArgs + result[end:]
		}
		offset = idx + len(prefix) + len(newArgs)
	}
	return result
}

// replaceInSelectorStrings replaces .orig with .token only inside querySelector strings.
func replaceInSelectorStrings(content, orig, token string) string {
	// Match querySelector('...') or querySelectorAll('...')
	re := regexp.MustCompile(`(querySelector(?:All)?\(\s*['"])([^'"]*)(['"]\s*\))`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		prefix, selector, suffix := parts[1], parts[2], parts[3]
		// Only replace .classname (CSS class selector syntax), never substrings in attribute selectors
		dotClassRe := regexp.MustCompile(`\.` + regexp.QuoteMeta(orig) + `\b`)
		selector = dotClassRe.ReplaceAllString(selector, "."+token)
		return prefix + selector + suffix
	})
}

// replaceInHTMLClassStrings replaces class names inside class="..." template strings.
func replaceInHTMLClassStrings(content, orig, token string) string {
	// Match class="..." (common in innerHTML/template literals)
	re := regexp.MustCompile(`class="([^"]*)"`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		classes := strings.Fields(parts[1])
		for i, cls := range classes {
			if cls == orig {
				classes[i] = token
			}
		}
		return `class="` + strings.Join(classes, " ") + `"`
	})
}

// replaceInClassNameAssignments replaces class names inside className="..." assignments.
func replaceInClassNameAssignments(content, orig, token string) string {
	// Match .className="..." or .className='...' (JS property assignment)
	re := regexp.MustCompile(`\.className\s*=\s*['"]([^'"]+)['"]`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		classes := strings.Fields(parts[1])
		for i, cls := range classes {
			if cls == orig {
				classes[i] = token
			}
		}
		// Preserve the original quote style
		quote := string(match[len(match)-1])
		return `.className=` + quote + strings.Join(classes, " ") + quote
	})
}

// replaceJSText finds inline <script> blocks in HTML output and applies JS replacements.
func replaceJSText(html string, mapping map[string]string) string {
	scriptRe := regexp.MustCompile(`(?is)(<script[^>]*>)(.*?)(</script>)`)
	return scriptRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := scriptRe.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		// Skip scripts with src= attribute
		if strings.Contains(parts[1], "src=") {
			return match
		}
		// Skip JSON-LD and import maps
		if strings.Contains(parts[1], "application/ld+json") || strings.Contains(parts[1], "importmap") {
			return match
		}
		return parts[1] + replaceJS(parts[2], mapping) + parts[3]
	})
}

// replaceCSSText finds inline <style> blocks in HTML output and applies CSS replacements.
func replaceCSSText(html string, mapping map[string]string) string {
	styleRe := regexp.MustCompile(`(?is)(<style[^>]*>)(.*?)(</style>)`)
	return styleRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := styleRe.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		return parts[1] + replaceCSS(parts[2], mapping) + parts[3]
	})
}

func applyRegexReplace(content string, mapping map[string]string) string {
	result := content
	for key, token := range mapping {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		typ, orig := parts[0], parts[1]
		switch typ {
		case "class":
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(orig) + `\b`)
			result = re.ReplaceAllString(result, token)
		case "id":
			re := regexp.MustCompile(`\bid="` + regexp.QuoteMeta(orig) + `"`)
			result = re.ReplaceAllString(result, `id="`+token+`"`)
		case "data":
			re := regexp.MustCompile(`\bdata-` + regexp.QuoteMeta(orig) + `\b`)
			result = re.ReplaceAllString(result, "data-"+token)
		}
	}
	return result
}
