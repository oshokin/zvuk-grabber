package utils

import (
	"bufio"
	"iter"
	"math"
	"math/rand/v2"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// ImageJPEGMimeType is the MIME type for JPEG images.
	ImageJPEGMimeType = "image/jpeg"

	// ImagePNGMimeType is the MIME type for PNG images.
	ImagePNGMimeType = "image/png"

	// textMimeTypePrefix is the MIME prefix used to detect text content types.
	textMimeTypePrefix = "text/"
	// applicationJSONMimeType is the MIME type for JSON payloads treated as text.
	applicationJSONMimeType = "application/json"
	// samlMetadataMimeTypePrefix is the MIME prefix for SAML metadata XML payloads.
	samlMetadataMimeTypePrefix = "application/samlmetadata+xml"

	// windowsReservedNameCON is the Windows reserved device name CON.
	windowsReservedNameCON = "CON"
	// windowsReservedNamePRN is the Windows reserved device name PRN.
	windowsReservedNamePRN = "PRN"
	// windowsReservedNameAUX is the Windows reserved device name AUX.
	windowsReservedNameAUX = "AUX"
	// windowsReservedNameNUL is the Windows reserved device name NUL.
	windowsReservedNameNUL = "NUL"
)

// invalidCharsPattern includes ASCII control characters (0-31) and Windows-restricted characters: < > : " / \ | ? *.
//
//nolint:gochecknoglobals // This is immutable, pre-compiled regex pattern and used as a constant.
var invalidCharsPattern = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

// SafeIntToUint8 converts an int value to an uint8 safely,
// ensuring that the value does not exceed the maximum limit of uint8.
func SafeIntToUint8(val int) uint8 {
	if val < 0 {
		return 0
	}

	if val > math.MaxUint8 {
		return math.MaxUint8
	}

	return uint8(val)
}

// SafeUint64ToInt64 converts a uint64 value to an int64 safely,
// ensuring that the value does not exceed the maximum limit of int64.
func SafeUint64ToInt64(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(val)
}

// SanitizeFilename sanitizes a filename or folder name to be valid on both Windows and Unix-like systems.
// It removes or replaces invalid characters, handles Windows reserved names, and ensures the filename is not empty.
func SanitizeFilename(name string) string {
	if name == "" {
		return ""
	}

	result := invalidCharsPattern.ReplaceAllString(name, "_")

	// Extract base filename (without extension) for comparison.
	baseName := result
	if dotIndex := strings.LastIndex(result, "."); dotIndex != -1 {
		baseName = result[:dotIndex]
	}

	// If base name is a Windows reserved name, prepend an underscore.
	if isWindowsReservedName(baseName) {
		result = "_" + result
	}

	// Remove trailing dots from the filename.
	result = strings.TrimRight(result, ".")

	// Ensure the filename is not empty.
	if result == "" {
		result = "_"
	}

	return result
}

// isWindowsReservedName reports whether the name is a Windows reserved device name.
func isWindowsReservedName(name string) bool {
	name = strings.ToUpper(name)
	switch name {
	case windowsReservedNameCON, windowsReservedNamePRN, windowsReservedNameAUX, windowsReservedNameNUL:
		return true
	}

	return len(name) == 4 && (strings.HasPrefix(name, "COM") || strings.HasPrefix(name, "LPT")) && name[3] >= '1' &&
		name[3] <= '9'
}

// RandomPause pauses execution for a random duration between min and max values.
// The min and max parameters should be of type time.Duration and represent
// the lower and upper bounds of the delay period, respectively.
func RandomPause(minPause, maxPause time.Duration) {
	// Ensure minPause is always less than or equal to maxPause.
	if minPause > maxPause {
		minPause, maxPause = maxPause, minPause
	}

	if minPause == maxPause {
		time.Sleep(minPause)

		return
	}

	// Generate a random duration between minPause and maxPause.
	randomDelay := minPause + time.Duration(
		//nolint:gosec // math/rand/v2 is secure.
		rand.Int64N(int64(maxPause-minPause)),
	)

	time.Sleep(randomDelay)
}

// SetFileExtension ensures the file has the specified extension.
// If the filename already has the correct extension, it is returned unchanged.
// If the filename has a different extension, the old extension is replaced with the new one.
// If the filename has no extension, the new extension is appended.
func SetFileExtension(filename, extension string, isExtensionReplaced bool) string {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	currentExt := filepath.Ext(filename)
	if currentExt == extension {
		return filename
	}

	if isExtensionReplaced {
		// Remove existing extension if present.
		filename = strings.TrimSuffix(filename, currentExt)
	}

	return filename + extension
}

// IsFileExist checks if a file exists at the specified path.
// It returns true if the file exists and is not a directory, false if the file does not exist,
// and an error if there was an issue accessing the file.
func IsFileExist(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return !stat.IsDir(), nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// ReadUniqueLinesFromFile reads a text file and returns a slice of unique non-empty lines.
// It skips empty lines and ensures that each line in the returned slice is unique.
func ReadUniqueLinesFromFile(path string) ([]string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var (
		uniqueLines = make(map[string]struct{})
		lines       []string
		scanner     = bufio.NewScanner(file)
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if _, exists := uniqueLines[line]; !exists {
			uniqueLines[line] = struct{}{}

			lines = append(lines, line)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// AddUnique adds value to the set and returns true when the value was inserted.
// The set map must be initialized by the caller.
func AddUnique[T comparable](set map[T]struct{}, value T) bool {
	if _, exists := set[value]; exists {
		return false
	}

	set[value] = struct{}{}

	return true
}

// ExtractNamedGroup extracts the value of a named capturing group from a regex match.
// It returns an empty string if the group is not found or if there is no match.
func ExtractNamedGroup(re *regexp.Regexp, groupName, input string) string {
	match := re.FindStringSubmatch(input)
	if match == nil {
		return ""
	}

	// Map group names to their corresponding values.
	for i, name := range re.SubexpNames() {
		if name == groupName {
			return match[i]
		}
	}

	return ""
}

// IsTextContentType checks if the given content type represents a text-based format.
// It supports common text content types like "text/*", "application/json", and "application/samlmetadata+xml".
// It also checks that the charset, if present, is either "utf-8" or "us-ascii".
func IsTextContentType(contentType string) bool {
	parsedType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	isText := (strings.HasPrefix(parsedType, textMimeTypePrefix) && len(parsedType) > len(textMimeTypePrefix)) ||
		parsedType == applicationJSONMimeType || strings.HasPrefix(parsedType, samlMetadataMimeTypePrefix)
	if !isText {
		return false
	}

	charset := strings.ToLower(params["charset"])

	return charset == "" || charset == "utf-8" || charset == "us-ascii"
}

// Map applies a transformation function to each element of a slice and returns a new slice with the results.
func Map[E, S any](v []E, transformFunc func(E) S) []S {
	result := make([]S, len(v))
	for i := range v {
		result[i] = transformFunc(v[i])
	}

	return result
}

// MapIterator applies a transformation function to each element of an iter.Seq sequence
// and returns a new slice with the results.
func MapIterator[E, S any](v iter.Seq[E], transformFunc func(E) S) []S {
	result := make([]S, 0)
	for i := range v {
		result = append(result, transformFunc(i))
	}

	return result
}
