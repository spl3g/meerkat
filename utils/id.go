package utils

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const IDSeparator = '|'

func ConcatIDs(pre string, post string) string {
	return fmt.Sprintf("%s%c%s", pre, IDSeparator, post)
}

func ConstructID(names ...string) string {
	var builder strings.Builder
	builder.Write([]byte(names[0]))
	for _, name := range names[1:] {
		builder.WriteByte(IDSeparator)
		builder.Write([]byte(name))
	}

	return builder.String()
}

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
	return fmt.Fprintf(w, "%c", IDSeparator)
}

type EntityID struct {
	Kind   string
	Labels map[string]string
}

func (e EntityID) Canonical() string {
	keys := make([]string, 0, len(e.Labels))
	for k := range e.Labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			printSep(&b)
		}
		formatKV(&b, k, e.Labels[k])
	}

	return b.String()
}
