package worker

import (
	"github.com/hack-pad/hackpad/internal/process"
)

type Remote struct {
}

type openFile struct {
	filePath   string
	seekOffset uint
}

func NewRemote(local *Local, pid process.PID, command string, args []string, attr *process.ProcAttr) (*Remote, error) {
	var openFiles []openFile
	for _, f := range attr.Files {
		info, err := local.process.Files().Fstat(f.FID)
		if err != nil {
			return nil, err
		}
		openFiles = append(openFiles, openFile{
			filePath:   info.Name(),
			seekOffset: 0, // TODO expose seek offset in file descriptor
		})
	}
	// TODO
	return &Remote{}, nil
}
