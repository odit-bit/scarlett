package cluster

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	clusterProto "github.com/odit-bit/scarlett/api/cluster/proto"
	storeproto "github.com/odit-bit/scarlett/store/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	credOpts grpc.DialOption
	pool     *pool
}

func NewClient() (*Client, error) {
	creds := insecure.NewCredentials()
	opts := grpc.WithTransportCredentials(creds)
	return &Client{
		credOpts: opts,
		pool: &pool{
			c:          map[string]*grpc.ClientConn{},
			leaderConn: nil,
			mx:         sync.Mutex{},
		},
	}, nil
}

func NewTLSClient(config *tls.Config) (*Client, error) {
	creds := credentials.NewServerTLSFromCert(&config.Certificates[0])
	opts := grpc.WithTransportCredentials(creds)
	return &Client{
		credOpts: opts,
	}, nil

}

func (c *Client) Close() error {
	c.pool.mx.Lock()
	defer c.pool.mx.Unlock()

	var err error
	for _, conn := range c.pool.c {
		err = errors.Join(conn.Close())
	}
	return err
}

func (c *Client) Join(ctx context.Context, joinAddr, nodeAddr, nodeID string) error {
	conn, err := c.pool.GetConn(ctx, joinAddr, c.credOpts)
	if err != nil {
		return err
	}
	defer conn.Close()

	lAddr, err := c.getLeaderAddr(ctx, conn)
	if err != nil {
		return err
	}

	lConn := conn
	if lAddr != joinAddr && lAddr != "" {
		lConn, err = grpc.DialContext(ctx, lAddr, c.credOpts)
		if err != nil {
			return err
		}
	}
	defer lConn.Close()

	cc := clusterProto.NewClusterClient(lConn)
	_, err = cc.Join(ctx, &clusterProto.JoinRequest{
		Address: nodeAddr,
		Id:      nodeID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getLeaderAddr(ctx context.Context, conn *grpc.ClientConn) (string, error) {
	cc := clusterProto.NewClusterClient(conn)
	resp, err := cc.GetNodeRaftAddr(ctx, &clusterProto.NodeRaftAddrRequest{})
	if err != nil {
		return "", err
	}

	return resp.Addr, nil
}

// return remote node http address in cluster
func (c *Client) GetNodeAPI(ctx context.Context, addr string) (string, error) {
	conn, err := c.pool.GetConn(ctx, addr, c.credOpts)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	cc := clusterProto.NewClusterClient(conn)
	res, err := cc.GetNodeApiUrl(ctx, &clusterProto.Request{}, grpc.WaitForReady(true))
	if err != nil {
		return "", err
	}
	return res.Url, nil
}

// client should get leader node addr automatically
func (c *Client) Set(ctx context.Context, addr string, key string, value string) (string, error) {
	conn, err := c.pool.GetConn(ctx, addr, c.credOpts)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// cc := clusterProto.NewClusterClient(conn)
	// resp, err := cc.GetNodeRaftAddr(ctx, &clusterProto.NodeRaftAddrRequest{})
	// if err != nil {
	// 	return "", err
	// }
	leaderAddr, err := c.getLeaderAddr(ctx, conn)
	if err != nil {
		return "", fmt.Errorf("failed get leader %v", err)
	}

	lConn := conn
	if leaderAddr != addr {
		lConn, err = c.pool.GetConn(ctx, leaderAddr, c.credOpts)
		if err != nil {
			return "", err
		}
	}
	defer lConn.Close()

	sc := storeproto.NewStorerClient(lConn)
	payload := storeproto.CmdRequestPayload{
		Cmd:   storeproto.Command_Type_Set,
		Key:   []byte(key),
		Value: []byte(value),
	}

	b, err := proto.Marshal(&payload)
	if err != nil {
		return "", err
	}

	cmd := storeproto.CmdRequest{
		Payload: b,
	}

	resp2, err := sc.Command(ctx, &cmd)
	if err != nil {
		return "", err
	}
	if resp2.Err != "" {
		return resp2.Err, nil
	}

	return resp2.Msg, nil

}

func (c *Client) Get(ctx context.Context, addr, key string) ([]byte, error) {
	conn, err := c.pool.GetConn(ctx, addr, c.credOpts)
	if err != nil {
		return nil, err
	}

	sc := storeproto.NewStorerClient(conn)
	bKey := []byte(key)
	resp, err := sc.Query(ctx, &storeproto.QueryRequest{
		Query: storeproto.QueryType_Get,
		Args:  [][]byte{bKey},
	})
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

type pool struct {
	c          map[string]*grpc.ClientConn
	leaderConn *grpc.ClientConn
	mx         sync.Mutex
}

func (p *pool) GetConn(ctx context.Context, addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	conn, ok := p.c[addr]
	if ok {
		state := conn.GetState()
		switch state {
		case connectivity.Connecting, connectivity.Idle, connectivity.Ready:
			return conn, nil
		case connectivity.Shutdown:
			conn.Close()
		}
	}

	inCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(inCtx, addr, opts...)
	if err != nil {
		conn.Close()
		delete(p.c, addr)
		return nil, err
	}
	p.c[addr] = conn
	return conn, err
}
