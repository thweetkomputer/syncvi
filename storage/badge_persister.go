package storage

type BadgePersister struct {
}

func (b *BadgePersister) Save(data []byte) error {
	return nil
}

func (b *BadgePersister) ReadSnapshot() []byte {
	return nil
}

func (b *BadgePersister) ReadRaftState() []byte {
	return nil
}

func (b *BadgePersister) RaftStateSize() int {
	return 0
}
