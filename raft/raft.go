package raft

import (
	"comp90020-assignment/raft/rpc"
	"comp90020-assignment/storage"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

const (
	LEADER = iota
	FOLLOWER
	CANDIDATE
)

type Raft struct {
	mu    sync.Mutex          // Lock to protect shared access to this peer's state
	peers []*raftpb.ClientEnd // RPC end points of all peers
	me    int32               // this peer's index into peers[]
	dead  int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int32
	votedFor    int32
	log         []LogEntry

	commitIndex int32
	lastApplied int32

	nextIndex  []int32
	matchIndex []int32

	state          int
	heartbeatTimer *time.Timer
	electionTimer  *time.Timer
	applyCh        chan ApplyMsg

	snapshot []byte

	commitCond *sync.Cond
	applyCond  *sync.Cond

	//rpcServer *RaftRpcServer
	raftpb.UnimplementedRaftServer
	persister storage.Persister
}

func (rf *Raft) mustEmbedUnimplementedRaftServer() {
	//TODO implement me
	panic("implement me")
}

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int32
	CommandTerm  int32

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int32
	SnapshotIndex int32
}

type LogEntry struct {
	Index   int32
	Term    int32
	Command *EditCommand
}

type Command interface {
	serialize() []byte
	deserialize([]byte)
}

type EditCommand struct {
}

func (c *EditCommand) serialize() []byte {
	return []byte{}
}

func (c *EditCommand) deserialize([]byte) {
	return
}

func (rf *Raft) GetState() (int32, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.state == LEADER
}

func (rf *Raft) persist() {
	// TODO: Implement this
}

func (rf *Raft) readPersist(data []byte) {
	// TODO: Implement this
}

func (rf *Raft) readSnapshot(index int, snapshot []byte) {
	// TODO: Implement this
}

func (rf *Raft) RequestVote(ctx context.Context, args *raftpb.RequestVoteRequest) (reply *raftpb.RequestVoteResponse, err error) {
	log.Printf("%d[%d] RequestVote %v", rf.me, rf.currentTerm, args)
	reply = &raftpb.RequestVoteResponse{}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if *args.Term < rf.currentTerm {
		reply.Term = &rf.currentTerm
		voteGranted := false
		reply.VoteGranted = &voteGranted
		return
	}
	if *args.Term > rf.currentTerm {
		rf.currentTerm = *args.Term
		rf.becomeFollower(true)
	}
	if (rf.votedFor == -1 || rf.votedFor == *args.CandidateId) &&
		(*args.LastLogTerm > rf.log[len(rf.log)-1].Term ||
			*args.LastLogTerm == rf.log[len(rf.log)-1].Term && *args.LastLogIndex >= rf.log[len(rf.log)-1].Index) {
		rf.votedFor = *args.CandidateId
		reply.Term = &rf.currentTerm
		voteGranted := true
		reply.VoteGranted = &voteGranted
		rf.electionTimer.Reset(time.Duration(rand.Intn(200)+300) * time.Millisecond)
		log.Printf("%d[%d] voted for %d", rf.me, rf.currentTerm, args.CandidateId)
		return
	}
	reply.Term = &rf.currentTerm
	voteGranted := false
	reply.VoteGranted = &voteGranted
	return
}

func (rf *Raft) SendRequestVote(peer int32, args *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error) {
	ctx := context.Background()
	return rf.peers[peer].SendRequestVote(ctx, args)
}

func (rf *Raft) AppendEntries(ctx context.Context, args *raftpb.AppendEntriesRequest) (reply *raftpb.AppendEntriesResponse, err error) {
	reply = &raftpb.AppendEntriesResponse{}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if *args.Term < rf.currentTerm {
		reply.Term = &rf.currentTerm
		success := false
		reply.Success = &success
		return
	}
	if *args.Term > rf.currentTerm {
		rf.currentTerm = *args.Term
		rf.becomeFollower(false)
	}
	rf.electionTimer.Reset(time.Duration(rand.Intn(200)+300) * time.Millisecond)
	if *args.PrevLogIndex > rf.log[len(rf.log)-1].Index ||
		*args.PrevLogIndex >= rf.log[0].Index &&
			*args.PrevLogTerm != rf.log[*args.PrevLogIndex-rf.log[0].Index].Term {
		reply.Term = &rf.currentTerm
		success := false
		reply.Success = &success
		log.Printf("%d[%d] reject", rf.me, rf.currentTerm)
		return
	}
	if *args.PrevLogIndex >= rf.log[0].Index {
		for i := 0; i < len(args.Entries); i++ {
			logIndex := *args.Entries[i].Index
			// TODO maybe error
			if logIndex > rf.log[len(rf.log)-1].Index {
				entries := make([]LogEntry, len(args.Entries[i:]))
				for j := 0; j < len(entries); j++ {
					command := &EditCommand{}
					command.deserialize(args.Entries[i+j].Command)
					entries[j] = LogEntry{
						Index:   *args.Entries[i+j].Index,
						Term:    *args.Entries[i+j].Term,
						Command: command,
					}
				}
				rf.log = append(rf.log, entries...)
				break
			}
			if rf.log[logIndex-rf.log[0].Index].Term != *args.Entries[i].Term {
				entries := make([]LogEntry, len(args.Entries[i:]))
				for j := 0; j < len(entries); j++ {
					command := &EditCommand{}
					command.deserialize(args.Entries[i+j].Command)
					entries[j] = LogEntry{
						Index:   *args.Entries[i+j].Index,
						Term:    *args.Entries[i+j].Term,
						Command: command,
					}
				}
				rf.log = append(rf.log[:logIndex-rf.log[0].Index], entries...)
				break
			}
		}
	}
	if *args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = *args.LeaderCommit
		if rf.commitIndex > rf.log[len(rf.log)-1].Index {
			rf.commitIndex = rf.log[len(rf.log)-1].Index
		}
		rf.applyCond.Broadcast()
	}
	reply.Term = &rf.currentTerm
	success := true
	reply.Success = &success
	return
}

func (rf *Raft) SendAppendEntries(peer int32, args *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error) {
	ctx := context.Background()
	return rf.peers[peer].SendAppendEntries(ctx, args)
}

func (rf *Raft) InstallSnapshot(ctx context.Context, args *raftpb.InstallSnapshotRequest) (reply *raftpb.InstallSnapshotResponse, err error) {
	reply = &raftpb.InstallSnapshotResponse{}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	reply.Term = &rf.currentTerm
	if *args.Term < rf.currentTerm {
		return
	}
	if *args.Term > rf.currentTerm {
		rf.currentTerm = *args.Term
		rf.becomeFollower(false)
	}
	rf.electionTimer.Reset(time.Duration(rand.Intn(200)+300) * time.Millisecond)
	if *args.LastIncludedIndex <= rf.log[0].Index {
		return
	}
	if *args.LastIncludedIndex < rf.log[len(rf.log)-1].Index {
		rf.log = append([]LogEntry{rf.log[*args.LastIncludedIndex-rf.log[0].Index]}, rf.log[*args.LastIncludedIndex+1-rf.log[0].Index:]...)
	} else {
		rf.log = []LogEntry{{*args.LastIncludedIndex, *args.LastIncludedTerm, &EditCommand{}}}
	}
	log.Printf("%d[%d] install snapshot %d", rf.me, rf.currentTerm, args.LastIncludedIndex)
	rf.snapshot = args.Data
	return
}

func (rf *Raft) SendInstallSnapshot(peer int32, args *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error) {
	ctx := context.Background()
	return rf.peers[peer].SendInstallSnapshot(ctx, args)
}

func (rf *Raft) becomeFollower(hb bool) {
	rf.state = FOLLOWER
	rf.heartbeatTimer.Stop()
	rf.votedFor = -1
	if hb {
		rf.electionTimer.Reset(time.Duration(rand.Intn(200)+300) * time.Millisecond)
	}
}

func (rf *Raft) becomeCandidate() {
	rf.state = CANDIDATE
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.electionTimer.Reset(time.Duration(rand.Intn(200)+300) * time.Millisecond)
	log.Printf("%d[%d] -> candidate %v", rf.me, rf.currentTerm, time.Now())
}

func (rf *Raft) becomeLeader() {
	rf.state = LEADER
	rf.electionTimer.Stop()
	rf.nextIndex = make([]int32, len(rf.peers))
	rf.matchIndex = make([]int32, len(rf.peers))
	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex[i] = rf.log[len(rf.log)-1].Index + 1
		rf.matchIndex[i] = 0
	}
	go rf.leaderCommit(rf.currentTerm)
	rf.heartbeatTimer.Reset(time.Duration(1) * time.Millisecond)
	log.Printf("%d[%d] -> leader %v", rf.me, rf.currentTerm, time.Now())
}

func (rf *Raft) Start(command []byte) (int32, int32, bool) {
	index := int32(-1)
	term := int32(-1)
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	term = rf.currentTerm
	isLeader = rf.state == LEADER
	if isLeader {
		cmd := &EditCommand{}
		cmd.deserialize(command)
		log := LogEntry{
			Index:   rf.log[len(rf.log)-1].Index + 1,
			Term:    rf.currentTerm,
			Command: cmd,
		}
		rf.log = append(rf.log, log)
		index = log.Index
		rf.heartbeatTimer.Reset(time.Duration(1) * time.Millisecond)
	}

	return index, term, isLeader
}

func (rf *Raft) leaderCommit(term int32) {
	for {
		rf.mu.Lock()
		if rf.state != LEADER || rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		for rf.log[len(rf.log)-1].Term < rf.currentTerm {
			rf.commitCond.Wait()
		}
		commitIndex := rf.commitIndex
		for i := rf.commitIndex + 1; i <= rf.log[len(rf.log)-1].Index; i++ {
			count := 1
			for j := int32(0); j < int32(len(rf.peers)); j++ {
				if j == rf.me {
					continue
				}
				if rf.matchIndex[j] >= i {
					count++
				}
			}
			if count > len(rf.peers)/2 {
				commitIndex = i
			}
		}
		if commitIndex > rf.commitIndex {
			rf.commitIndex = commitIndex
			rf.applyCond.Broadcast()
			log.Printf("%d[%d] commit %d", rf.me, rf.currentTerm, rf.commitIndex)
		}
		rf.mu.Unlock()
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func (rf *Raft) broadcast(term int32) {
	var wg sync.WaitGroup
	wg.Add(len(rf.peers) - 1)
	for i := int32(0); i < int32(len(rf.peers)); i++ {
		if i == rf.me {
			continue
		}
		go func(server int32) {
			defer wg.Done()
			rf.mu.Lock()
			if rf.state != LEADER || term != rf.currentTerm {
				rf.mu.Unlock()
				return
			}
			if rf.log[0].Index < rf.nextIndex[server] {
				entries := make([]LogEntry, len(rf.log[rf.nextIndex[server]-rf.log[0].Index:]))
				copy(entries, rf.log[rf.nextIndex[server]-rf.log[0].Index:])
				prevLogIndex := rf.nextIndex[server] - 1
				prefLogTerm := rf.log[prevLogIndex-rf.log[0].Index].Term
				pbEntries := make([]*raftpb.Entry, len(entries))
				for i := 0; i < len(entries); i++ {
					command := entries[i].Command.serialize()
					pbEntries[i] = &raftpb.Entry{
						Index:   &entries[i].Index,
						Term:    &entries[i].Term,
						Command: command,
					}
				}
				args := &raftpb.AppendEntriesRequest{
					Term:         &term,
					LeaderId:     &rf.me,
					PrevLogIndex: &prevLogIndex,
					PrevLogTerm:  &prefLogTerm,
					Entries:      pbEntries,
					LeaderCommit: &rf.commitIndex,
				}
				if len(args.Entries) > 0 {
					log.Printf("%d[%d] -> %d %v", rf.me, rf.currentTerm, server, args.Entries)
				}
				rf.mu.Unlock()
				reply, err := rf.SendAppendEntries(server, args)
				if err != nil {
					return
				}
				rf.mu.Lock()
				if term != rf.currentTerm {
					rf.mu.Unlock()
					return
				}
				if *reply.Term > rf.currentTerm {
					rf.currentTerm = *reply.Term
					rf.becomeFollower(true)
				} else if *reply.Success {
					if len(args.Entries) > 0 && *args.Entries[len(args.Entries)-1].Index > rf.matchIndex[server] {
						rf.matchIndex[server] = *args.Entries[len(args.Entries)-1].Index
						rf.nextIndex[server] = rf.matchIndex[server] + 1
						rf.commitCond.Broadcast()
					}
				} else {
					rf.nextIndex[server] = (rf.nextIndex[server] + rf.matchIndex[server]) / 2
					if rf.nextIndex[server] < 1 {
						rf.nextIndex[server] = 1
					}
				}
				rf.mu.Unlock()
			} else {
				// TODO snapshot
				args := &raftpb.InstallSnapshotRequest{
					Term:              &term,
					LeaderId:          &rf.me,
					LastIncludedIndex: &rf.log[0].Index,
					LastIncludedTerm:  &rf.log[0].Term,
					Data:              rf.persister.ReadSnapshot(),
				}
				rf.mu.Unlock()
				reply, err := rf.SendInstallSnapshot(server, args)
				if err != nil {
					return
				}
				rf.mu.Lock()
				if term != rf.currentTerm {
					rf.mu.Unlock()
					return
				}
				if *reply.Term > rf.currentTerm {
					rf.currentTerm = *reply.Term
					rf.becomeFollower(true)
				} else {
					if *args.LastIncludedIndex+1 > rf.nextIndex[server] {
						rf.nextIndex[server] = *args.LastIncludedIndex + 1
					}
					if *args.LastIncludedIndex > rf.matchIndex[server] {
						rf.matchIndex[server] = *args.LastIncludedIndex
						rf.commitCond.Broadcast()
					}
				}
				rf.mu.Unlock()
			}
		}(i)
	}
	// TODO wg.Wait() maybe become follower again
}

func (rf *Raft) elect(term int32) {
	var wg sync.WaitGroup
	wg.Add(len(rf.peers) - 1)
	agree := 1
	rf.mu.Lock()
	args := &raftpb.RequestVoteRequest{
		Term:         &term,
		CandidateId:  &rf.me,
		LastLogIndex: &rf.log[len(rf.log)-1].Index,
		LastLogTerm:  &rf.log[len(rf.log)-1].Term,
	}
	rf.mu.Unlock()
	for i := int32(0); i < int32(len(rf.peers)); i++ {
		if i == rf.me {
			continue
		}
		go func(server int32) {
			defer wg.Done()
			reply, err := rf.SendRequestVote(server, args)
			if err != nil {
				return
			}
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if term != rf.currentTerm {
				return
			}
			if *reply.VoteGranted {
				agree++
				if agree > len(rf.peers)/2 {
					if rf.state == CANDIDATE && rf.currentTerm == term {
						rf.becomeLeader()
					}
				}
			} else if *reply.Term > rf.currentTerm {
				rf.currentTerm = *reply.Term
				rf.becomeFollower(true)
			}
		}(i)
	}
	if agree > len(rf.peers)/2 && rf.state == CANDIDATE && rf.currentTerm == term {
		rf.becomeLeader()
	}
	wg.Wait()
}

func (rf *Raft) ticker() {
	for {
		select {
		case <-rf.electionTimer.C:
			rf.mu.Lock()
			if rf.state == LEADER {
				rf.electionTimer.Stop()
				rf.mu.Unlock()
				continue
			}
			rf.becomeCandidate()
			go rf.elect(rf.currentTerm)
			rf.mu.Unlock()
		case <-rf.heartbeatTimer.C:
			rf.mu.Lock()
			if rf.state == LEADER {
				rf.heartbeatTimer.Reset(time.Duration(100) * time.Millisecond)
				go rf.broadcast(rf.currentTerm)
			} else {
				rf.heartbeatTimer.Stop()
			}
			rf.mu.Unlock()
		}
	}
}

//func newServer(rf *Raft) raftpb.RaftServer {
//	return &RaftRpcServer{
//		Raft: rf,
//	}
//}

func Make(peers []*raftpb.ClientEnd, me int32,
	persister storage.Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.log = make([]LogEntry, 1)
	rf.applyCh = applyCh
	rf.electionTimer = time.NewTimer(time.Duration(rand.Intn(150)+150) * time.Millisecond)
	rf.heartbeatTimer = time.NewTimer(time.Duration(100) * time.Millisecond)
	rf.becomeFollower(false)
	rf.commitCond = sync.NewCond(&rf.mu)
	rf.applyCond = sync.NewCond(&rf.mu)

	// initialize from state persisted before a crash
	rf.readPersist(rf.persister.ReadRaftState())
	rf.snapshot = rf.persister.ReadSnapshot()
	ip := peers[me].Ip
	port := peers[me].Port
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	raftpb.RegisterRaftServer(grpcServer, rf)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		os.Exit(1)
	}()
	log.Printf("Server started at %s:%s", ip, port)

	go rf.ticker()
	go func() {
		for {
			rf.mu.Lock()
			for rf.commitIndex <= rf.lastApplied {
				rf.applyCond.Wait()
			}
			if rf.lastApplied >= rf.log[0].Index {
				applyMsgs := make([]ApplyMsg, rf.commitIndex-rf.lastApplied)
				for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
					applyMsg := ApplyMsg{
						CommandValid: true,
						Command:      rf.log[i-rf.log[0].Index].Command,
						CommandIndex: i,
						CommandTerm:  rf.log[i-rf.log[0].Index].Term,
					}
					applyMsgs[i-rf.lastApplied-1] = applyMsg
				}
				rf.mu.Unlock()
				for i := 0; i < len(applyMsgs); i++ {
					applyCh <- applyMsgs[i]
					rf.mu.Lock()
					rf.lastApplied++
					rf.mu.Unlock()
				}
			} else {
				log.Printf("%d[%d] snapshot %v", rf.me, rf.currentTerm, rf.snapshot)
				log.Printf("%d[%d] log %v %v", rf.me, rf.currentTerm, rf.log, rf.lastApplied)
				applyMsg := ApplyMsg{
					CommandValid:  false,
					SnapshotValid: true,
					Snapshot:      rf.snapshot,
					SnapshotTerm:  rf.log[0].Term,
					SnapshotIndex: rf.log[0].Index,
				}
				rf.mu.Unlock()
				applyCh <- applyMsg
				rf.mu.Lock()
				rf.lastApplied = applyMsg.SnapshotIndex
				rf.mu.Unlock()
			}
			rf.mu.Lock()
			log.Printf("%d[%d] applied %v", rf.me, rf.currentTerm, rf.lastApplied)
			rf.mu.Unlock()
		}
	}()

	return rf
}