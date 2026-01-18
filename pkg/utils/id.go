package utils

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const IDSeparator = "|"

var EmptyNameError = errors.New("'name' is required")

func CheckName(name string) error {
	if len(name) == 0 {
		return EmptyNameError
	}

	return nil
}

func formatKV(w io.Writer, key string, value string) (int, error) {
	return fmt.Fprintf(w, "%s=%s", key, value)
}

func printSep(w io.Writer) (int, error) {
	return fmt.Fprintf(w, "%s", IDSeparator)
}

type EntityID struct {
	Kind   string
	Labels map[string]string
}

func (e EntityID) Canonical() string {
	keys := make([]string, 0, len(e.Labels)+1)
	for k := range e.Labels {
		keys = append(keys, k)
	}
	keys = append(keys, "kind")

	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			printSep(&b)
		}
		if k == "kind" {
			formatKV(&b, k, e.Kind)
			continue
		}
		formatKV(&b, k, e.Labels[k])
	}

	return b.String()
}

func ParseEntityID(str string) EntityID {
	e := EntityID{
		Kind:   "",
		Labels: make(map[string]string),
	}

	if str == "" {
		return e
	}

	labels := strings.SplitSeq(str, IDSeparator)
	for label := range labels {
		kv := strings.Split(label, "=")
		if len(kv) < 2 {
			continue
		}
		if kv[0] == "kind" {
			e.Kind = kv[1]
			continue
		}
		e.Labels[kv[0]] = kv[1]
	}
	return e
}

