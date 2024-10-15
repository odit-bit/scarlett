package store

import (
	"context"

	"github.com/odit-bit/scarlett/store/storepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ storepb.CommandServer = (*nodeRpcHandler)(nil)

// var _ clusterpb.ClusterServer = (*nodeRpcHandler)(nil)

type NodeService interface {
	Command(ctx context.Context, cmd CMDType, args ...[]byte) (*CommandResponse, error)
	Get(ctx context.Context, key []byte) ([]byte, bool, error)
}

type ClusterSevice interface {
	Join(id, addr string) error
	Leader() (id, address string)
	Remove(id string) error
}

var _ storepb.ClusterServer = (*nodeRpcHandler)(nil)

type nodeRpcHandler struct {
	node    NodeService
	cluster ClusterSevice
	storepb.UnimplementedCommandServer
	storepb.UnimplementedClusterServer
}

// Join implements storepb.ClusterServer.
func (n *nodeRpcHandler) Join(_ context.Context, req *storepb.JoinRequest) (*storepb.JoinResponse, error) {
	if err := n.cluster.Join(req.Id, req.Address); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "node-handler: %v", err.Error())
	}
	return &storepb.JoinResponse{Msg: "success"}, nil
}

func (n *nodeRpcHandler) RemoveNode(_ context.Context, req *storepb.RemoveRequest) (*storepb.RemoveResponse, error) {
	if err := n.cluster.Remove(req.Id); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "node-handler: %v", err.Error())
	}
	return &storepb.RemoveResponse{
		Msg: "ok",
	}, nil
}

// Leader implements storepb.ClusterServer.
func (n *nodeRpcHandler) Leader(context.Context, *storepb.LeaderRequest) (*storepb.LeaderResponse, error) {
	id, address := n.cluster.Leader()
	if address == "" {
		return nil, status.Errorf(codes.NotFound, "leader not found")
	}
	return &storepb.LeaderResponse{
		Id:   id,
		Addr: address,
	}, nil
}

// Get implements storepb.CommandServer.
func (n *nodeRpcHandler) Get(ctx context.Context, req *storepb.GetRequest) (*storepb.GetResponse, error) {
	v, ok, err := n.node.Get(ctx, req.Key)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "key not found")
	}
	return &storepb.GetResponse{
		Value: v,
	}, nil
}

func NewNodeRpcHandler(ns NodeService) *nodeRpcHandler {
	return &nodeRpcHandler{
		node:                       ns,
		UnimplementedCommandServer: storepb.UnimplementedCommandServer{},
	}
}

// Delete implements storepb.CommandServer.
func (n *nodeRpcHandler) Delete(ctx context.Context, req *storepb.DeleteRequest) (*storepb.DeleteResponse, error) {
	if req.Cmd != storepb.Command_Type_Delete {
		return nil, status.Errorf(codes.InvalidArgument, "invalid command type %T", req.Cmd.Descriptor())
	}

	if v, err := n.node.Command(ctx, DELETE_CMD_TYPE, req.Key); err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	} else {
		return &storepb.DeleteResponse{
			Message: v.Message,
		}, nil

	}
}

// Set implements storepb.CommandServer.
func (n *nodeRpcHandler) Set(ctx context.Context, req *storepb.SetRequest) (*storepb.SetResponse, error) {
	if req.Cmd != storepb.Command_Type_Set {
		return nil, status.Errorf(codes.InvalidArgument, "invalid command type %T", req.Cmd.Descriptor())
	}

	if v, err := n.node.Command(ctx, SET_CMD_TYPE, req.Key, req.Value); err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	} else {
		return &storepb.SetResponse{
			Message: v.Message,
		}, nil
	}

}

// // GetNodeApiUrl implements proto.ClusterServer.
// func (n *nodeRpcHandler) GetNodeApiUrl(context.Context, *clusterpb.Request) (*clusterpb.Response, error) {
// 	return nil, status.Error(codes.Unimplemented, "deprecated")
// 	// return &clusterpb.Response{
// 	// 	Url: n.store.httpAddr,
// 	// }, nil
// }

// // GetNodeRaftAddr implements proto.ClusterServer.
// func (n *nodeRpcHandler) GetNodeRaftAddr(context.Context, *clusterpb.NodeRaftAddrRequest) (*clusterpb.NodeRaftAddrResponse, error) {
// 	// return &clusterpb.NodeRaftAddrResponse{
// 	// 	Addr: n.cs.Addr(),
// 	// }, nil
// 	return nil, status.Error(codes.Unimplemented, "deprecated")
// }

// // Join implements proto.ClusterServer.
// func (n *nodeRpcHandler) Join(ctx context.Context, req *clusterpb.JoinRequest) (*clusterpb.JoinResponse, error) {
// 	if err := n.node.AddNode(req.Id, req.Address); err != nil {
// 		return nil, status.Error(codes.Unavailable, err.Error())
// 	}
// 	return &clusterpb.JoinResponse{
// 		Msg: "success",
// 	}, nil
// }
