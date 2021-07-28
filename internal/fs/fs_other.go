// +build !js

package fs

type persistFs struct {
	hackpadfs.FS
}

func newPersistDB(name string, relaxedDurability bool, shouldCache ShouldCacher) (*persistFs, error) {
	panic("not implemented")
}

func (p *persistFs) Clear() error {
	panic("not implemented")
}
