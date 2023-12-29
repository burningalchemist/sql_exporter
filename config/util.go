package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

func checkCollectorRefs(collectorRefs []string, ctx string) error {
	// At least one collector, no duplicates
	if len(collectorRefs) == 0 {
		return fmt.Errorf("no collectors defined for %s", ctx)
	}
	for i, ci := range collectorRefs {
		for _, cj := range collectorRefs[i+1:] {
			if ci == cj {
				return fmt.Errorf("duplicate collector reference %q in %s", ci, ctx)
			}
		}
	}
	return nil
}

func resolveCollectorRefs(
	collectorRefs []string, collectors map[string]*CollectorConfig, ctx string,
) ([]*CollectorConfig, error) {
	resolved := make([]*CollectorConfig, 0, len(collectorRefs))
	found := make(map[*CollectorConfig]bool)
	for _, cref := range collectorRefs {
		cref_resolved := false
		for k, c := range collectors {
			matched, err := filepath.Match(cref, k)
			if err != nil {
				return nil, fmt.Errorf("bad collector %q referenced in %s: %w", cref, ctx, err)
			}
			if matched && !found[c] {
				resolved = append(resolved, c)
				found[c] = true
				cref_resolved = true
			}
		}
		if !cref_resolved {
			return nil, fmt.Errorf("unknown collector %q referenced in %s", cref, ctx)
		}
	}
	klog.Infof("Resolved collectors for %s: %v", ctx, len(resolved))
	return resolved, nil
}

func checkLabel(label string, ctx ...string) error {
	if label == "" {
		return fmt.Errorf("empty label defined in %s", strings.Join(ctx, " "))
	}
	if label == "job" || label == TargetLabel {
		return fmt.Errorf("reserved label %q redefined in %s", label, strings.Join(ctx, " "))
	}
	return nil
}

func checkOverflow(m map[string]any, ctx string) error {
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown fields in %s: %s", ctx, strings.Join(keys, ", "))
	}
	return nil
}
