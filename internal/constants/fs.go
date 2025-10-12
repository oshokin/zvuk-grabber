package constants

import "os"

const (
	// DefaultFilePermissions sets the default permissions for regular files: (rw-r--r--).
	// Owner: read and write;
	// Group: read;
	// Others: read.
	DefaultFilePermissions os.FileMode = 0o644

	// DefaultFolderPermissions sets the default permissions for regular folders: (rwxr-xr-x).
	// Owner: read, write, and execute;
	// Group: read and execute;
	// Others: read and execute.
	DefaultFolderPermissions os.FileMode = 0o755
)

// File extension constants.
const (
	ExtensionMP3  = ".mp3"
	ExtensionFLAC = ".flac"
	ExtensionBin  = ".bin"
)
