package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	heartbeatv1 "github.com/twflem/dist-sys/raft/gen/proto/raft/v1"
	"github.com/twflem/dist-sys/raft/gen/proto/raft/v1/heartbeatv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	connect "github.com/bufbuild/connect-go"
)

const address = "0.0.0.0:8080"

func main() {
	fmt.Println("beginning raft")
	go Raft()

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

func Raft() {
	peers := scaffoldPeers()
	heartbeat := time.NewTicker(time.Millisecond * 1000)
	for {
		<-heartbeat.C
		go func() {
			for i := range peers {
				_, err := peers[i].heartbeatClient.Beat(context.Background(), connect.NewRequest(&heartbeatv1.BeatRequest{
					SenderName: os.Getenv("HOSTNAME"),
				}))
				if err != nil {
					fmt.Printf("failed to sent heartbeat to peer %s with err: %s\n", peers[i].name, err)
				}
			}
		}()
	}
}

type Peer struct {
	name            string
	heartbeatClient heartbeatv1connect.HeartbeatClient
}

func scaffoldPeers() []Peer {
	self := os.Getenv("HOSTNAME")
	peerHostnameBase := strings.Split(self, "-")[0]
	numReplicasStr := os.Getenv("NUM_REPLICAS")
	numReplicas, err := strconv.Atoi(numReplicasStr)
	if err != nil {
		panic(fmt.Errorf("could not parse NUM_REPLICAS as int: %s", err.Error()))
	}
	var peers []Peer
	for i := 0; i < numReplicas; i++ {
		baseHostName := fmt.Sprintf("http://%s-%d.%s.default.svc.cluster.local:8080", peerHostnameBase, i, peerHostnameBase)
		if baseHostName != self {
			peers = append(peers, Peer{
				name:            baseHostName,
				heartbeatClient: heartbeatv1connect.NewHeartbeatClient(http.DefaultClient, baseHostName),
			})
		}
	}
	return peers
}

type HeartbeatService struct {
	heartbeatv1connect.UnimplementedHeartbeatHandler
}

func (h *HeartbeatService) Beat(_ context.Context, req *connect.Request[heartbeatv1.BeatRequest]) (*connect.Response[heartbeatv1.BeatResponse], error) {
	fmt.Printf("Hearbeat received from %s\n", req.Msg.SenderName)
	return connect.NewResponse(&heartbeatv1.BeatResponse{}), nil
}
