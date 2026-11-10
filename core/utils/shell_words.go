package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// stringToShellWordsHook turns a string config value into an argv slice for
// []string targets. Environment overrides arrive as strings, so
// `AUTH_TOKEN_STORE_COMMAND_GET="pass show lazyspotify/token"` has to become
// three arguments rather than one binary name with spaces.
func stringToShellWordsHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to != reflect.TypeOf([]string{}) {
		return data, nil
	}
	return splitShellWords(data.(string))
}

// stringToIntSliceHook accepts comma-separated ints from environment
// overrides, for example AUTH_TOKEN_STORE_COMMAND_NOT_FOUND_EXIT_CODES=1,3.
func stringToIntSliceHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to != reflect.TypeOf([]int{}) {
		return data, nil
	}
	parts := strings.Split(data.(string), ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", part)
		}
		values = append(values, value)
	}
	return values, nil
}

// splitShellWords splits value on unquoted whitespace. Single quotes keep
// their contents verbatim; double quotes keep whitespace and backslash escapes.
func splitShellWords(value string) ([]string, error) {
	var (
		words   []string
		current strings.Builder
		inWord  bool
		quote   rune
		escaped bool
	)

	for _, r := range value {
		switch {
		case escaped:
			current.WriteRune(r)
			inWord = true
			escaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			default:
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == '\\':
			escaped = true
			inWord = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inWord {
				words = append(words, current.String())
				current.Reset()
				inWord = false
			}
		default:
			current.WriteRune(r)
			inWord = true
		}
	}

	if escaped || quote != 0 {
		return nil, fmt.Errorf("unbalanced quotes or escape in %q", value)
	}
	if inWord {
		words = append(words, current.String())
	}
	return words, nil
}
