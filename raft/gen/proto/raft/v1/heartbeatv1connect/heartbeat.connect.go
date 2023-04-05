// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/raft/v1/heartbeat.proto

package heartbeatv1connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v1 "github.com/twflem/dist-sys/raft/gen/proto/raft/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// HeartbeatName is the fully-qualified name of the Heartbeat service.
	HeartbeatName = "heartbeat.v1.Heartbeat"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// HeartbeatBeatProcedure is the fully-qualified name of the Heartbeat's Beat RPC.
	HeartbeatBeatProcedure = "/heartbeat.v1.Heartbeat/Beat"
)

// HeartbeatClient is a client for the heartbeat.v1.Heartbeat service.
type HeartbeatClient interface {
	Beat(context.Context, *connect_go.Request[v1.BeatRequest]) (*connect_go.Response[v1.BeatResponse], error)
}

// NewHeartbeatClient constructs a client for the heartbeat.v1.Heartbeat service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewHeartbeatClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) HeartbeatClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &heartbeatClient{
		beat: connect_go.NewClient[v1.BeatRequest, v1.BeatResponse](
			httpClient,
			baseURL+HeartbeatBeatProcedure,
			opts...,
		),
	}
}

// heartbeatClient implements HeartbeatClient.
type heartbeatClient struct {
	beat *connect_go.Client[v1.BeatRequest, v1.BeatResponse]
}

// Beat calls heartbeat.v1.Heartbeat.Beat.
func (c *heartbeatClient) Beat(ctx context.Context, req *connect_go.Request[v1.BeatRequest]) (*connect_go.Response[v1.BeatResponse], error) {
	return c.beat.CallUnary(ctx, req)
}

// HeartbeatHandler is an implementation of the heartbeat.v1.Heartbeat service.
type HeartbeatHandler interface {
	Beat(context.Context, *connect_go.Request[v1.BeatRequest]) (*connect_go.Response[v1.BeatResponse], error)
}

// NewHeartbeatHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewHeartbeatHandler(svc HeartbeatHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(HeartbeatBeatProcedure, connect_go.NewUnaryHandler(
		HeartbeatBeatProcedure,
		svc.Beat,
		opts...,
	))
	return "/heartbeat.v1.Heartbeat/", mux
}

// UnimplementedHeartbeatHandler returns CodeUnimplemented from all methods.
type UnimplementedHeartbeatHandler struct{}

func (UnimplementedHeartbeatHandler) Beat(context.Context, *connect_go.Request[v1.BeatRequest]) (*connect_go.Response[v1.BeatResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("heartbeat.v1.Heartbeat.Beat is not implemented"))
}
