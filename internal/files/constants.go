package files

import "os"

const (
	// DefaultFilePermissions sets regular file permissions to rw-r--r--.
	DefaultFilePermissions os.FileMode = 0o644

	// DefaultFolderPermissions sets directory permissions to rwxr-xr-x.
	DefaultFolderPermissions os.FileMode = 0o755
)

const (
	// ExtensionMP3 is the file extension for MP3 audio files.
	ExtensionMP3 = ".mp3"
	// ExtensionFLAC is the file extension for FLAC audio files.
	ExtensionFLAC = ".flac"
	// ExtensionBin is the file extension for unknown/binary audio payloads.
	ExtensionBin = ".bin"
)
