package context

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultRoot       = "./memory"
	projectMemoryFile = "project.md"
	rulesDir          = "rules"
)

// Bundle 是一次 memory 加载后的聚合结果。
type Bundle struct {
	Project string
	Rules   []Rule
}

// Rule 表示一个按路径匹配的 memory 规则文件。
type Rule struct {
	Name    string
	Paths   []string
	Content string
}

// Loader 负责从 memory 目录读取 project memory 和 rule memory。
type Loader struct {
	root string
}

// NewLoader 创建一个基于目录的 memory loader。
func NewLoader(root string) *Loader {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultRoot
	}
	return &Loader{root: root}
}

// Load 按给定路径加载 project memory 与命中的规则文件。
func (l *Loader) Load(paths []string) (Bundle, error) {
	project, err := l.loadProject()
	if err != nil {
		return Bundle{}, err
	}

	rules, err := l.loadRules(paths)
	if err != nil {
		return Bundle{}, err
	}

	return Bundle{
		Project: project,
		Rules:   rules,
	}, nil
}

// loadProject 加载全局 project memory；文件不存在时视为可选。
func (l *Loader) loadProject() (string, error) {
	content, err := os.ReadFile(filepath.Join(l.root, projectMemoryFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// loadRules 加载并筛选所有匹配路径条件的规则文件。
func (l *Loader) loadRules(paths []string) ([]Rule, error) {
	rulesRoot := filepath.Join(l.root, rulesDir)
	entries := make([]string, 0, 8)

	err := filepath.WalkDir(rulesRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	sort.Strings(entries)

	matched := make([]Rule, 0, len(entries))
	for _, entry := range entries {
		rule, err := loadRuleFile(entry)
		if err != nil {
			return nil, err
		}
		if !rule.matches(paths) {
			continue
		}
		matched = append(matched, rule)
	}

	return matched, nil
}

// loadRuleFile 读取单个规则文件并解析 front matter。
func loadRuleFile(filePath string) (Rule, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return Rule{}, err
	}

	paths, content := parseRuleFile(string(raw))
	return Rule{
		Name:    strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		Paths:   paths,
		Content: strings.TrimSpace(content),
	}, nil
}

// parseRuleFile 从 markdown 文本中提取 front matter 的 paths 和正文。
func parseRuleFile(raw string) ([]string, string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return nil, raw
	}

	end := strings.Index(raw[4:], "\n---\n")
	if end < 0 {
		return nil, raw
	}

	frontMatter := raw[4 : 4+end]
	content := raw[4+end+5:]
	return parseRulePaths(frontMatter), content
}

// parseRulePaths 解析 front matter 中的 paths 列表。
func parseRulePaths(frontMatter string) []string {
	lines := strings.Split(frontMatter, "\n")
	paths := make([]string, 0, 4)
	inPaths := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "paths:":
			inPaths = true
		case inPaths && strings.HasPrefix(trimmed, "- "):
			pattern := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if pattern != "" {
				paths = append(paths, normalizePattern(pattern))
			}
		case trimmed == "":
		default:
			inPaths = false
		}
	}

	return paths
}

// matches 判断规则是否命中当前任务传入的路径集合。
func (r Rule) matches(paths []string) bool {
	if len(r.Paths) == 0 {
		return true
	}
	if len(paths) == 0 {
		return false
	}

	for _, candidate := range paths {
		normalized := normalizePattern(candidate)
		for _, pattern := range r.Paths {
			if matchPattern(pattern, normalized) {
				return true
			}
		}
	}
	return false
}

// normalizePattern 把路径统一成可比较的斜杠格式。
func normalizePattern(v string) string {
	return strings.Trim(strings.ReplaceAll(v, "\\", "/"), "/")
}

// matchPattern 使用轻量 glob 语义匹配规则路径与候选路径。
func matchPattern(pattern, value string) bool {
	pattern = normalizePattern(pattern)
	value = normalizePattern(value)

	re := regexp.QuoteMeta(pattern)
	re = strings.ReplaceAll(re, "\\*\\*", ".*")
	re = strings.ReplaceAll(re, "\\*", "[^/]*")
	re = strings.ReplaceAll(re, "\\?", "[^/]")

	ok, err := regexp.MatchString("^"+re+"$", value)
	if err != nil {
		return false
	}
	return ok
}
