package storer

type QueueGetter interface {
	QueueGetFileRecord(path string, dest *FileRecord) <-chan error
}

func QueueGetFileRecord(s Storer, path string, dest *FileRecord) <-chan error {
	if queueSetter, ok := s.(QueueGetter); ok {
		return queueSetter.QueueGetFileRecord(path, dest)
	}

	err := make(chan error, 1)
	err <- s.GetFileRecord(path, dest)
	close(err)
	return err
}

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
