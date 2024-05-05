package editor

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"sync"
	editorpb "syncvi/editor/rpc"
	"syncvi/raft"
	raftpb "syncvi/raft/rpc"
)

type Server struct {
	mu      sync.Mutex
	me      int32
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	msgCh   chan interface{}

	maxraftstate int // snapshot if log grows this big

	data        []rune
	ops         map[int32]int64
	logTerm     map[int32]int32
	persister   raft.Persister
	commitIndex int32
	commitCond  *sync.Cond
	editorpb.UnimplementedEditorServer
}

type Op struct {
	Diff []byte
	CkId int32
	Seq  int64
}

func (op Op) serialize() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(op)
	if err != nil {
		log.Fatalf("Error encoding op: %v", err)
	}
	return buf.Bytes()
}

func deserializeOp(data []byte) Op {
	var op Op
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&op)
	if err != nil {
		log.Fatalf("Error decoding op: %v", err)
	}
	return op

}

const (
	OK             = "OK"
	ErrWrongLeader = "ErrWrongLeader"
	ErrRepeated    = "ErrRepeated"
)

func (s *Server) processOp() {
	for {
		msg := <-s.applyCh
		if msg.CommandValid {
			s.mu.Lock()
			var op Op
			if len(msg.Command) > 0 {
				op = deserializeOp(msg.Command)
				if s.ops[op.CkId] >= op.Seq {
					s.commitCond.Broadcast()
					s.mu.Unlock()
					continue
				}
			}
			dif := op.Diff
			if len(dif) > 0 {
				s.data = patch(s.data, dif)
			}
			s.msgCh <- struct{}{}
			s.logTerm[msg.CommandIndex] = msg.CommandTerm
			s.commitIndex = msg.CommandIndex
			s.ops[op.CkId] = op.Seq
			if s.maxraftstate != -1 && s.persister.RaftStateSize() > s.maxraftstate {
				w := new(bytes.Buffer)
				e := gob.NewEncoder(w)
				e.Encode(s.data)
				e.Encode(s.ops)
				s.rf.Snapshot(s.commitIndex, w.Bytes())
			}
			s.commitCond.Broadcast()
			s.mu.Unlock()
		} else if msg.SnapshotValid {
			s.mu.Lock()
			snapshot := msg.Snapshot
			r := bytes.NewBuffer(snapshot)
			d := gob.NewDecoder(r)
			if d.Decode(&s.data) != nil {
				log.Fatal("decode error")
			}
			s.commitIndex = msg.SnapshotIndex
			s.mu.Unlock()
			s.msgCh <- struct{}{}
		}
	}
}

func (s *Server) Do(ctx context.Context, req *editorpb.DoRequest) (*editorpb.DoResponse, error) {
	resp := &editorpb.DoResponse{}
	e := OK
	resp.Err = &e
	op := Op{
		Diff: req.Diff,
		CkId: *req.CkID,
		Seq:  *req.Seq,
	}

	index, term, isLeader := s.rf.Start(op.serialize())
	if !isLeader {
		e := ErrWrongLeader
		resp.Err = &e
		return resp, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ops[*req.CkID] >= *req.Seq {
		return resp, nil
	}
	for s.commitIndex < index {
		s.commitCond.Wait()
	}
	defer delete(s.logTerm, index)
	if term != s.logTerm[index] {
		e := ErrRepeated
		resp.Err = &e
		return resp, nil
	}
	return resp, nil
}

func StartServer(peers []*raftpb.ClientEnd, nodes []*ClientEnd, me int32,
	persister raft.Persister, msgCh chan interface{}) *Server {
	applyCh := make(chan raft.ApplyMsg)
	s := &Server{
		me:           me,
		rf:           raft.Make(peers, me, persister, applyCh),
		applyCh:      applyCh,
		msgCh:        msgCh,
		maxraftstate: 1024,
		data:         make([]rune, 0),
		ops:          make(map[int32]int64),
		logTerm:      make(map[int32]int32),
		persister:    persister,
		commitIndex:  0,
	}
	snapshot := s.persister.ReadSnapshot()
	if snapshot != nil && len(snapshot) > 0 {
		r := bytes.NewBuffer(snapshot)
		d := gob.NewDecoder(r)
		var data []rune
		var ops map[int32]int64
		if d.Decode(&data) != nil || d.Decode(&ops) != nil {
			log.Fatal("decode error")
		} else {
			s.data = data
			s.ops = ops
		}
	}
	s.mu = sync.Mutex{}
	s.commitCond = sync.NewCond(&s.mu)
	ip := nodes[me].Ip
	port := nodes[me].Port
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	editorpb.RegisterEditorServer(grpcServer, s)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		os.Exit(1)
	}()
	go s.processOp()
	return s
}
