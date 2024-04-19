package raft

type Persister interface {
	// Save the data to the storage
	Save(raftstate []byte, snapshot []byte)
	ReadSnapshot() []byte
	ReadRaftState() []byte
	RaftStateSize() int
}
