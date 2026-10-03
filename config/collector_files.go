package config

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// DefaultCollectorPathPattern matches .yml / .yaml collector filenames (case-insensitive).
const DefaultCollectorPathPattern = `(?i)\.(yml|yaml)$`

// collectorSources returns the ordered list of collector file sources to resolve.
// Precedence for the dedicated path: CLI override (-config.collector.path) wins over
// collector_path from the config file / SQLEXPORTER_COLLECTOR_PATH.
func (c *Config) collectorSources() []string {
	sources := make([]string, 0, len(c.CollectorFiles)+1)

	path := c.CollectorPath
	if CollectorPathOverride != "" {
		path = CollectorPathOverride
	}
	if path != "" {
		sources = append(sources, path)
	}
	sources = append(sources, c.CollectorFiles...)
	return sources
}

// collectorPathRecursiveEnabled returns whether directory sources should be scanned recursively.
// Default is true. CLI override wins over config/env when set.
func (c *Config) collectorPathRecursiveEnabled() bool {
	if CollectorPathRecursiveOverride != nil {
		return *CollectorPathRecursiveOverride
	}
	if c.CollectorPathRecursive != nil {
		return *c.CollectorPathRecursive
	}
	return true
}

// collectorPathFilePattern returns the regex used to select filenames under directories.
// Default is DefaultCollectorPathPattern. CLI override wins over config/env when set.
func (c *Config) collectorPathFilePattern() (string, error) {
	pattern := c.CollectorPathPattern
	if CollectorPathPatternOverride != "" {
		pattern = CollectorPathPatternOverride
	}
	if pattern == "" {
		pattern = DefaultCollectorPathPattern
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return "", fmt.Errorf("invalid collector_path_pattern %q: %w", pattern, err)
	}
	return pattern, nil
}

// loadCollectorFiles resolves collector_path and collector_files entries to concrete files
// and loads the collectors they define.
//
// Files discovered from a directory (collector_path or a directory entry in collector_files)
// are loaded only when their collector_name matches target/jobs collectors references.
// Explicit globs/files in collector_files still load every matching file.
func (c *Config) loadCollectorFiles() error {
	baseDir := filepath.Dir(c.configFile)
	configFileAbs := absPathOrSelf(c.configFile)
	refs := c.referencedCollectorPatterns()
	recursive := c.collectorPathRecursiveEnabled()

	pattern, err := c.collectorPathFilePattern()
	if err != nil {
		return err
	}
	fileRE, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	for _, source := range c.collectorSources() {
		files, fromDirectory, err := resolveCollectorFiles(baseDir, source, recursive, fileRE)
		if err != nil {
			return err
		}
		slog.Debug("External collector files found", "count", len(files), "source", source,
			"directory", fromDirectory, "recursive", recursive, "pattern", pattern)

		for _, file := range files {
			// Avoid treating the main exporter config as a collector when a directory
			// that also contains sql_exporter.yml is used as a source.
			if absPathOrSelf(file) == configFileAbs {
				slog.Debug("Skipping main configuration file while loading collectors", "file", file)
				continue
			}
			if err := c.loadCollectorFile(file, fromDirectory, refs); err != nil {
				return err
			}
		}
	}

	return nil
}

// referencedCollectorPatterns returns collector name patterns from target and jobs.
func (c *Config) referencedCollectorPatterns() []string {
	refs := make([]string, 0)
	if c.Target != nil {
		refs = append(refs, c.Target.CollectorRefs...)
	}
	for _, job := range c.Jobs {
		refs = append(refs, job.CollectorRefs...)
	}
	return refs
}

// collectorNameRequested reports whether name matches any collector reference pattern.
func collectorNameRequested(name string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return true, nil
	}
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("bad collector reference %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// resolveCollectorFiles turns a source (glob, file, or directory) into a sorted list of files.
// Relative sources are resolved against baseDir (the directory of the main config file).
// fromDirectory is true when at least one resolved path was a directory that was expanded.
func resolveCollectorFiles(
	baseDir, source string, recursive bool, fileRE *regexp.Regexp,
) (files []string, fromDirectory bool, err error) {
	if source == "" {
		return nil, false, nil
	}

	pattern := source
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(baseDir, pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		// The only error filepath.Glob returns is a bad pattern.
		return nil, false, fmt.Errorf("error resolving collector files for %s: %w", source, err)
	}

	files = make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, false, fmt.Errorf("error accessing collector path %s: %w", match, err)
		}
		if info.IsDir() {
			dirFiles, err := listCollectorFiles(match, recursive, fileRE)
			if err != nil {
				return nil, false, err
			}
			files = append(files, dirFiles...)
			fromDirectory = true
			continue
		}
		files = append(files, match)
	}

	sort.Strings(files)
	return uniqueStrings(files), fromDirectory, nil
}

// listCollectorFiles returns sorted files under dir whose basename matches fileRE.
// When recursive is true, subdirectories are walked. Symlink directories are not followed
// (filepath.WalkDir default). Symlink files are included if their names match.
func listCollectorFiles(dir string, recursive bool, fileRE *regexp.Regexp) ([]string, error) {
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("error reading collector directory %s: %w", dir, err)
		}
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !fileRE.MatchString(entry.Name()) {
				continue
			}
			files = append(files, filepath.Join(dir, entry.Name()))
		}
		sort.Strings(files)
		return files, nil
	}

	files := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !fileRE.MatchString(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking collector directory %s: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

// collectorNameCandidatesFromFilename derives possible collector names from a filename.
// Examples:
//
//	pricing.collector.yml -> pricing.collector, pricing
//	pricing.yml           -> pricing
func collectorNameCandidatesFromFilename(path string) []string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	candidates := []string{name}
	if strings.EqualFold(filepath.Ext(name), ".collector") {
		candidates = append(candidates, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	return uniqueStrings(candidates)
}

// filenameMatchesCollectorRefs reports whether a collector file should be opened based on
// its filename matching target/jobs collector references. This avoids reading/parsing
// unrelated files in a shared collectors directory (including broken YAML).
func filenameMatchesCollectorRefs(path string, refs []string) (bool, error) {
	if len(refs) == 0 {
		return true, nil
	}
	for _, candidate := range collectorNameCandidatesFromFilename(path) {
		requested, err := collectorNameRequested(candidate, refs)
		if err != nil {
			return false, err
		}
		if requested {
			return true, nil
		}
	}
	return false, nil
}

// loadCollectorFile reads a single collector definition file and appends it to c.Collectors.
// When fromDirectory is true, collectors that are not referenced by target/jobs are skipped
// by filename first (so unused/broken files are never opened), then by collector_name.
func (c *Config) loadCollectorFile(path string, fromDirectory bool, refs []string) error {
	if fromDirectory {
		requested, err := filenameMatchesCollectorRefs(path, refs)
		if err != nil {
			return err
		}
		if !requested {
			slog.Debug("Skipping collector file not matching target/jobs refs", "file", path)
			return nil
		}
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// For directory sources, also honor the collector_name inside the file.
	if fromDirectory {
		var peek struct {
			Name string `yaml:"collector_name"`
		}
		if err := yaml.Unmarshal(buf, &peek); err != nil {
			return fmt.Errorf("error parsing collector file %s: %w", path, err)
		}
		if peek.Name == "" {
			return fmt.Errorf("collector file %s must define a collector with a name", path)
		}
		requested, err := collectorNameRequested(peek.Name, refs)
		if err != nil {
			return err
		}
		if !requested {
			slog.Debug("Skipping collector not referenced by target/jobs", "name", peek.Name, "file", path)
			return nil
		}
	}

	var node yaml.Node
	if err := yaml.Unmarshal(buf, &node); err != nil {
		return fmt.Errorf("error parsing collector file %s: %w", path, err)
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return fmt.Errorf("collector file %s is not a valid YAML document", path)
	}

	top := node.Content[0]
	if top.Kind != yaml.MappingNode {
		return fmt.Errorf("collector file %s must define a single YAML map/object at the top level", path)
	}

	// Each external file must define one collector object, not a collectors list.
	for i := 0; i < len(top.Content); i += 2 {
		keyNode := top.Content[i]
		valNode := top.Content[i+1]
		if keyNode.Value == "collectors" && valNode.Kind == yaml.SequenceNode {
			return fmt.Errorf(
				"collector file %s contains a 'collectors' list. Each file must define a single collector object",
				path,
			)
		}
	}

	cc := CollectorConfig{}
	if err := node.Decode(&cc); err != nil {
		return fmt.Errorf("error parsing collector file %s: %w", path, err)
	}
	if cc.Name == "" {
		return fmt.Errorf("collector file %s must define a collector with a name", path)
	}

	c.Collectors = append(c.Collectors, &cc)
	slog.Debug("Loaded collector", "name", cc.Name, "file", path)
	return nil
}

func absPathOrSelf(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func uniqueStrings(in []string) []string {
	if len(in) < 2 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
