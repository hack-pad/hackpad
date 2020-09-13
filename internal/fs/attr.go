package fs

// Attr defines file descriptor inheritance rules for a new set of descriptors
// Ignore is unsupported.
// Pipe will create a new pipe and attach it to the child process.
// FID will inherit that descriptor in the child process.
type Attr struct {
	Ignore bool
	Pipe   bool
	FID    FID
}
