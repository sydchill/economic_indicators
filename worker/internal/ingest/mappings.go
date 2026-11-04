package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type MappingFile struct {
	Version  int       `json:"version"`
	Mappings []Mapping `json:"mappings"`
}

type Mapping struct {
	Match  string `json:"match"`
	Suffix string `json:"suffix"`
}

type CompiledMapping struct {
	Pattern *regexp.Regexp
	Suffix  string
}

// LoadMapping parses seed/indicator_mapping.json into compiled regexes.
// First-match-wins, so more specific patterns must come before generic ones.
func LoadMapping(path string) ([]CompiledMapping, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var mf MappingFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&mf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if mf.Version != 1 {
		return nil, fmt.Errorf("unsupported mapping version %d", mf.Version)
	}

	out := make([]CompiledMapping, 0, len(mf.Mappings))
	for _, m := range mf.Mappings {
		re, err := regexp.Compile(m.Match)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", m.Match, err)
		}
		out = append(out, CompiledMapping{Pattern: re, Suffix: m.Suffix})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no mappings defined", path)
	}
	return out, nil
}

func ResolveSuffix(mappings []CompiledMapping, eventName string) (string, bool) {
	for _, m := range mappings {
		if m.Pattern.MatchString(eventName) {
			return m.Suffix, true
		}
	}
	return "", false
}
