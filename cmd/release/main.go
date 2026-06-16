// release – automated release helper for the Hugo theme.
//
// Responsibilities:
//   - Parses --version=<semver> flag
//   - Detects previous git tag matching v<semver>
//   - Collects commits between previous tag and HEAD
//   - Categorizes commits using conventional commit prefixes
//   - Generates/updates CHANGELOG.md
//   - Updates theme.toml version
//   - Optionally tags and pushes (--tag --push)
//
// Usage:
//
//	go run ./cmd/release --version=0.2.0
//	go run ./cmd/release --version=0.2.0 --tag --push --dry-run
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type commit struct {
	hash    string
	message string
}

type category struct {
	name    string
	commits []commit
}

var catOrder = []string{
	"Added", "Changed", "Fixed", "Removed",
	"Performance", "Security", "Docs", "Chore", "Tests", "Other",
}

var convPrefix = map[string]string{
	"feat":     "Added",
	"fix":      "Fixed",
	"perf":     "Performance",
	"docs":     "Docs",
	"chore":    "Chore",
	"refactor": "Changed",
	"test":     "Tests",
	"build":    "Chore",
	"ci":       "Chore",
	"style":    "Changed",
	"revert":   "Changed",
	"security": "Security",
}

func main() {
	version := flag.String("version", "", "Semantic version (e.g. 0.2.0)")
	dryRun := flag.Bool("dry-run", false, "Print changes without writing files")
	tag := flag.Bool("tag", false, "Create git tag")
	push := flag.Bool("push", false, "Push tag to origin")
	flag.Parse()

	if *version == "" {
		fmt.Fprintln(os.Stderr, "Usage: release --version=<semver> [--dry-run] [--tag] [--push]")
		os.Exit(1)
	}

	semver := strings.TrimPrefix(*version, "v")

	// Find previous tag
	prevTag := findPrevTag("v" + semver)
	var prevRef string
	if prevTag != "" {
		prevRef = prevTag
	} else {
		// Fallback: first commit
		prevRef = firstCommit()
	}

	if prevRef == "" {
		fmt.Fprintln(os.Stderr, "Could not determine previous reference point")
		os.Exit(1)
	}

	// Collect commits
	commits := collectCommits(prevRef, "HEAD")

	// Categorize
	cats := categorize(commits)

	// Generate changelog entry
	entry := generateEntry(semver, cats)

	if *dryRun {
		fmt.Println("=== CHANGELOG entry ===")
		fmt.Print(entry)
		fmt.Println("=== end ===")
		return
	}

	// Update CHANGELOG.md
	updateChangelog(entry)

	// Update theme.toml version
	updateThemeVersion(semver)

	fmt.Printf("Release %s prepared.\n", semver)

	if *tag {
		tagName := "v" + semver
		run("git", "tag", "-a", tagName, "-m", "Release "+tagName)
		fmt.Printf("Tag %s created.\n", tagName)

		if *push {
			run("git", "push", "origin", tagName)
			fmt.Printf("Tag %s pushed.\n", tagName)
		}
	}
}

func findPrevTag(currentTag string) string {
	out, err := exec.Command("git", "tag", "--sort=-v:refname").Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if re.MatchString(t) && t != currentTag && "v"+t != currentTag {
			return t
		}
	}
	return ""
}

func firstCommit() string {
	out, err := exec.Command("git", "rev-list", "--max-parents=0", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func collectCommits(from, to string) []commit {
	rangeSpec := from + ".." + to
	out, err := exec.Command("git", "log", "--no-merges", "--format=%H %s", rangeSpec).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "git log failed: %v\n", err)
		return nil
	}

	var commits []commit
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, commit{hash: parts[0], message: parts[1]})
	}

	// Reverse to chronological order
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits
}

func categorize(commits []commit) []category {
	catMap := make(map[string][]commit)
	re := regexp.MustCompile(`^(\w+)(?:\([^)]*\))?!?:\s*(.*)$`)

	for _, c := range commits {
		m := re.FindStringSubmatch(c.message)
		if m != nil {
			prefix := strings.ToLower(m[1])
			desc := m[2]
			catName, ok := convPrefix[prefix]
			if !ok {
				catName = "Other"
			}
			catMap[catName] = append(catMap[catName], commit{hash: c.hash, message: desc})
		} else {
			catMap["Other"] = append(catMap["Other"], c)
		}
	}

	var cats []category
	for _, name := range catOrder {
		if commits, ok := catMap[name]; ok && len(commits) > 0 {
			sort.Slice(commits, func(i, j int) bool {
				return strings.ToLower(commits[i].message) < strings.ToLower(commits[j].message)
			})
			cats = append(cats, category{name: name, commits: commits})
		}
	}

	return cats
}

func generateEntry(version string, cats []category) string {
	var b strings.Builder
	date := time.Now().Format("2006-01-02")
	fmt.Fprintf(&b, "## [%s] - %s\n\n", version, date)

	for _, cat := range cats {
		fmt.Fprintf(&b, "### %s\n\n", cat.name)
		for _, c := range cat.commits {
			fmt.Fprintf(&b, "- %s (%s)\n", c.message, c.hash[:7])
		}
		b.WriteString("\n")
	}

	return b.String()
}

func updateChangelog(entry string) {
	const path = "CHANGELOG.md"
	existing, err := os.ReadFile(path)
	if err != nil {
		// Create new changelog
		content := "# Changelog\n\n" + entry
		os.WriteFile(path, []byte(content), 0644)
		return
	}

	content := string(existing)
	// Find the first "## [" header and insert before it
	re := regexp.MustCompile(`(?m)^##\s+\[`)
	idx := re.FindStringIndex(content)
	if idx != nil {
		newContent := content[:idx[0]] + entry + content[idx[0]:]
		os.WriteFile(path, []byte(newContent), 0644)
	} else {
		// Append after "# Changelog" header or at end
		newContent := content + "\n" + entry
		os.WriteFile(path, []byte(newContent), 0644)
	}
}

func updateThemeVersion(version string) {
	const path = "theme.toml"
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read %s: %v\n", path, err)
		return
	}

	re := regexp.MustCompile(`(?m)^version\s*=\s*".*"`)
	newContent := re.ReplaceAll(content, []byte(`version = "`+version+`"`))
	os.WriteFile(path, newContent, 0644)
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %s %v: %v\n", name, args, err)
		os.Exit(1)
	}
}
