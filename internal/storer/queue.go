package storer

type QueueSetter interface {
	QueueSetFileRecord(path string, data *FileRecord) <-chan error
}

func QueueSetFileRecord(s Storer, path string, data *FileRecord) <-chan error {
	if queueSetter, ok := s.(QueueSetter); ok {
		return queueSetter.QueueSetFileRecord(path, data)
	}

	err := make(chan error, 1)
	err <- s.SetFileRecord(path, data)
	close(err)
	return err
}
