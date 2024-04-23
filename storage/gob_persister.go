package storage

import (
	"io"
	"log"
	"os"
	"sync"
)

type GoBPersister struct {
	mu            sync.Mutex
	raftstatePath string
	snapshotPath  string
}

func MakeGoBPersister(path string) *GoBPersister {
	if os.MkdirAll(path, os.ModePerm) != nil {
		log.Fatalf("Error creating directory: %v", path)
	}
	raftstatePath := path + "/raftstate"
	snapshotPath := path + "/snapshot"
	return &GoBPersister{
		raftstatePath: raftstatePath,
		snapshotPath:  snapshotPath,
	}
}

func (ps *GoBPersister) ReadRaftState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ReadFromFile(ps.raftstatePath)
}

func (ps *GoBPersister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ReadFromFile(ps.raftstatePath))
}

func (ps *GoBPersister) Save(raftstate []byte, snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// save to disk
	WriteToFile(ps.raftstatePath, raftstate)
	WriteToFile(ps.snapshotPath, snapshot)
}

func ReadFromFile(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading from file: %v", err)
	}
	return data
}

func WriteToFile(path string, data []byte) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	// flush
	err = file.Sync()
	if err != nil {
		log.Fatalf("Error flushing file: %v", err)
	}
}

func (ps *GoBPersister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ReadFromFile(ps.snapshotPath)
}

func (ps *GoBPersister) SnapshotSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ReadFromFile(ps.snapshotPath))
}
