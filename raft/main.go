package main

import (
	"context"
	"fmt"
	"net/http"

	heartbeatv1 "github.com/twflem/dist-sys/raft/gen/proto/raft/v1"
	"github.com/twflem/dist-sys/raft/gen/proto/raft/v1/heartbeatv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	connect "github.com/bufbuild/connect-go"
)

const address = "localhost:8080"

func main() {
	mux := http.NewServeMux()
	path, handler := heartbeatv1connect.NewHeartbeatHandler(&HeartbeatService{})
	mux.Handle(path, handler)
	fmt.Println("... Listening on", address)
	http.ListenAndServe(
		address,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}

type HeartbeatService struct {
	heartbeatv1connect.UnimplementedHeartbeatHandler
}

func (h *HeartbeatService) ReceiveHeartbeat(_ context.Context, _ *connect.Request[heartbeatv1.HeartbeatRequest]) (*connect.Response[heartbeatv1.HeartbeatResponse], error) {
	fmt.Println("Hearbeat received")
	return connect.NewResponse(&heartbeatv1.HeartbeatResponse{}), nil
}
