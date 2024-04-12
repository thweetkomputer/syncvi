package raftpb

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"strings"
)

type ClientEnd struct {
	Ip     string
	Port   string
	Client RaftClient
}

func (c *ClientEnd) SendRequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	if c.Client == nil {
		conn, err := grpc.Dial(c.Ip+":"+c.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Error dialing %s:%s: %v\n", c.Ip, c.Port, err)
			return nil, err
		}
		c.Client = NewRaftClient(conn)
	}
	return c.Client.RequestVote(ctx, req)
}

func (c *ClientEnd) SendAppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	if c.Client == nil {
		conn, err := grpc.Dial(c.Ip+":"+c.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		c.Client = NewRaftClient(conn)
	}
	return c.Client.AppendEntries(ctx, req)
}

func (c *ClientEnd) SendInstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error) {
	if c.Client == nil {
		conn, err := grpc.Dial(c.Ip+":"+c.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		c.Client = NewRaftClient(conn)
	}
	return c.Client.InstallSnapshot(ctx, req)
}

func MakeClientEnd(addr string) (end *ClientEnd) {
	addrParts := strings.Split(addr, ":")
	end = new(ClientEnd)
	end.Ip = addrParts[0]
	end.Port = addrParts[1]
	conn, err := grpc.Dial(addr)
	if err == nil {
		end.Client = NewRaftClient(conn)
	}
	return
}

func ParseClientEnd(peers string) []*ClientEnd {
	peersArr := strings.Split(peers, ",")
	clientEnds := make([]*ClientEnd, len(peersArr))
	for i, peer := range peersArr {
		clientEnds[i] = MakeClientEnd(peer)
	}
	return clientEnds
}
