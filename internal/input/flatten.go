package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Flatten expands CLI arguments and text files into one deduplicated URL list.
// Empty lines and lines starting with '#' are ignored to make URL lists convenient
// for manual editing.
func Flatten(args []string) ([]string, error) {
	seen := make(map[string]struct{}, len(args))
	result := make([]string, 0, len(args))

	for _, arg := range args {
		value := strings.TrimSpace(arg)
		if value == "" {
			continue
		}

		info, err := os.Stat(value)
		if err == nil && !info.IsDir() {
			lines, appendErr := appendFileLines(value, seen, result)
			if appendErr != nil {
				return nil, appendErr
			}

			result = lines

			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result, nil
}

// appendFileLines reads unique non-empty lines from a text file and appends them to result.
func appendFileLines(path string, seen map[string]struct{}, result []string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("failed to open URL list %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if _, ok := seen[line]; ok {
			continue
		}

		seen[line] = struct{}{}
		result = append(result, line)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return result, fmt.Errorf("failed to read URL list %q: %w", path, scanErr)
	}

	return result, nil
}
