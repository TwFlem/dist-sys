package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	raftv1 "github.com/twflem/dist-sys/raft/gen/proto/raft/v1"
	"github.com/twflem/dist-sys/raft/gen/proto/raft/v1/raftv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	connect "github.com/bufbuild/connect-go"
)

const address = "0.0.0.0:8080"
const termPath = "/data/termCounter"

func main() {
	var term int
	b, err := os.ReadFile(termPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}

		err := os.WriteFile(termPath, []byte("0"), fs.FileMode(os.O_WRONLY))
		if err != nil {
			panic(err)
		}
	}

	term, err = strconv.Atoi(string(b))
	if err != nil {
		panic(err)
	}

	self := NewSelf(term)
	go self.StartTermWatcher()
	go self.StartFollowing()

	mux := http.NewServeMux()
	path, handler := raftv1connect.NewRaftHandler(self)
	mux.Handle(path, handler)
	fmt.Println("... Listening on", address)
	http.ListenAndServe(
		address,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)

}

const (
	receiveEntryTimeoutUpper = 300
	receiveEntryTimeoutLower = 150
	electionTimeoutUpper     = 300
	electionTimeoutLower     = 150
)

type Peer struct {
	name       string
	raftClient raftv1connect.RaftClient
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
				name:       baseHostName,
				raftClient: raftv1connect.NewRaftClient(http.DefaultClient, baseHostName),
			})
		}
	}
	return peers
}

type Self struct {
	peers             []Peer
	term              int
	updateTermChan    chan int
	termMut           *sync.Mutex
	entryReceivedChan chan EntryReceivedEvent
	raftv1connect.UnimplementedRaftHandler
}

type EntryReceivedEvent struct {
	receivedTerm int
}

func NewSelf(previouslyPersistedTerm int) *Self {
	entryRecievedChan := make(chan EntryReceivedEvent)
	termChan := make(chan int)
	return &Self{
		peers:             scaffoldPeers(),
		entryReceivedChan: entryRecievedChan,
		term:              previouslyPersistedTerm,
		updateTermChan:    termChan,
	}
}

func (s *Self) StartFollowing() {
	ticker := time.NewTicker(randDur(receiveEntryTimeoutLower, receiveEntryTimeoutUpper))
	for {
		select {
		case e := <-s.entryReceivedChan:
			if e.receivedTerm > s.term {
				s.updateTermChan <- e.receivedTerm
			}
			ticker.Reset(randDur(receiveEntryTimeoutLower, receiveEntryTimeoutUpper))
		case <-ticker.C:
			s.StartElection()
			ticker.Reset(randDur(receiveEntryTimeoutLower, receiveEntryTimeoutUpper))
		}
	}
}

func (s *Self) StartElection() {
	majority := int(math.Ceil(float64(len(s.peers)) / 2.0))
	ticker := time.NewTicker(0)
Election:
	for {
		select {
		case e := <-s.entryReceivedChan:
			if e.receivedTerm > s.term {
				s.updateTermChan <- e.receivedTerm
				break Election
			}
		case <-ticker.C:
			voteSuccessChan := make(chan bool)
			voteErrChan := make(chan error)
			s.updateTermChan <- s.term + 1
			go func() {
				for i := range s.peers {
					_, err := s.peers[i].raftClient.Vote(context.Background(), connect.NewRequest(&raftv1.VoteRequest{
						SenderName: os.Getenv("HOSTNAME"),
					}))
					if err != nil {
						fmt.Printf("failed to sent heartbeat to peer %s with err: %s\n", s.peers[i].name, err)
						voteErrChan <- err
						return
					}
					voteSuccessChan <- true
				}
			}()
			successes := 0
			for i := 0; i < len(s.peers); i++ {
				select {
				case <-voteSuccessChan:
					successes += 1
				case err := <-voteErrChan:
					if obsErr, ok := MaybeGetObsoleteTermError(err); ok {
						s.updateTermChan <- int(obsErr.LatestTerm)
						break Election
					}
				}
			}
			if successes >= majority {
				s.StartLeading()
			}
			ticker.Reset(randDur(electionTimeoutLower, electionTimeoutUpper))
		}

	}
}

func (s *Self) StartLeading() {
	for {

	}
}

func (s *Self) AppendEntry(_ context.Context, req *connect.Request[raftv1.AppendEntryRequest]) (*connect.Response[raftv1.AppendEntryResponse], error) {
	fmt.Printf("Hearbeat received from %s\n", req.Msg.SenderName)
	if int(req.Msg.Term) < s.term {
		return nil, GetObsoleteTermError(s.term)
	}
	s.entryReceivedChan <- EntryReceivedEvent{
		receivedTerm: int(req.Msg.Term),
	}
	return connect.NewResponse(&raftv1.AppendEntryResponse{}), nil
}

func (s *Self) Vote(_ context.Context, req *connect.Request[raftv1.VoteRequest]) (*connect.Response[raftv1.VoteResponse], error) {
	fmt.Printf("Vote received from %s\n", req.Msg.SenderName)
	if int(req.Msg.Term) < s.term {
		return nil, GetObsoleteTermError(s.term)
	}
	return connect.NewResponse(&raftv1.VoteResponse{}), nil
}

func GetObsoleteTermError(latestTerm int) *connect.Error {
	err := connect.NewError(connect.CodeInvalidArgument, errors.New("obsolete term detected"))
	obseleteTermErr := raftv1.ObsoleteTermError{
		LatestTerm: int64(latestTerm),
	}
	detail, newErrDetailErr := connect.NewErrorDetail(&obseleteTermErr)
	if newErrDetailErr != nil {
		fmt.Println(newErrDetailErr)
	}
	err.AddDetail(detail)
	return err
}

func MaybeGetObsoleteTermError(err error) (*raftv1.ObsoleteTermError, bool) {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return nil, false
	}
	for _, detail := range connectErr.Details() {
		msg, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if obsoleteTermErr, ok := msg.(*raftv1.ObsoleteTermError); ok {
			return obsoleteTermErr, true
		}
	}
	return nil, false
}

func randDur(lower, upper int) time.Duration {
	return time.Duration(randInt(lower, upper))
}
func randInt(lower, upper int) int {
	return lower + rand.Intn(upper-lower)
}

func (s *Self) StartTermWatcher() {
	for {
		term := <-s.updateTermChan
		if term <= s.term {
			continue
		}
		s.term = term
		err := os.WriteFile(termPath, []byte(strconv.Itoa(term)), fs.FileMode(os.O_WRONLY))
		if err != nil {
			fmt.Println("could persist updated term")
			panic(err)
		}
	}
}
