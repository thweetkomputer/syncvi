package raft

type Persister interface {
	// Save the data to the storage
	Save(data []byte) error
	ReadSnapshot() []byte
	ReadRaftState() []byte
	RaftStateSize() int
}
