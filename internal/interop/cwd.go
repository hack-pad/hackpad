package interop

var currentWorkingDirectory = "/home/me"

func WorkingDirectory() string {
	return currentWorkingDirectory
}

// SetWorkingDirectory sets the current working directory to 'path'
// MUST be called from internal/process.Chdir. Does not perform any file existence checks.
func SetWorkingDirectory(path string) {
	currentWorkingDirectory = path
}
