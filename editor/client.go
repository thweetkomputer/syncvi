package editor

import (
	"comp90020-assignment/editor/rpc"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"time"
)

type Client struct {
	servers    []*ClientEnd
	leader     int32
	id         int32
	requestSeq int64
}

type ClientEnd struct {
	Ip     string
	Port   string
	Client editorpb.EditorClient
}

func MakeClient(servers []*ClientEnd, me int32) *Client {
	ck := new(Client)
	ck.servers = servers
	ck.id = me
	ck.requestSeq = 0
	return ck
}

func (ck *Client) Do(ctx context.Context, diff []byte) {
	ck.requestSeq++
	args := &editorpb.DoRequest{
		Diff: diff,
		CkID: &ck.id,
		Seq:  &ck.requestSeq,
	}
	for {
		resCh := make(chan interface{})
		leader := ck.leader
		var reply *editorpb.DoResponse
		var err error
		go func() {
			reply, err = ck.servers[leader].Do(ctx, args)
			if err == nil {
				resCh <- struct{}{}
			}
		}()
		select {
		case <-resCh:
			switch *reply.Err {
			case OK:
				return
			case ErrRepeated:
				continue
			}
		case <-time.After(1000 * time.Millisecond):
		}
		ck.leader = (ck.leader + 1) % int32(len(ck.servers))
	}
}

func (c *ClientEnd) Do(ctx context.Context, req *editorpb.DoRequest) (*editorpb.DoResponse, error) {
	if c.Client == nil {
		conn, err := grpc.Dial(c.Ip+":"+c.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		c.Client = editorpb.NewEditorClient(conn)
	}
	return c.Client.Do(ctx, req)
}

func MakeClientEnd(addr string) (end *ClientEnd) {
	addrParts := strings.Split(addr, ":")
	end = new(ClientEnd)
	end.Ip = addrParts[0]
	end.Port = addrParts[1]
	conn, err := grpc.Dial(addr)
	if err == nil {
		end.Client = editorpb.NewEditorClient(conn)
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
