package signaller

import (
	"fmt"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	pbion "github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
)

type GRPCConn struct {
	conf     GRPCConnConfig
	conn     *nrpc.Client
	onBroken func(error)
}

type GRPCConnConfig struct {
	NewConn func(service, peerNID string) (*nrpc.Client, error)
	DstNode *pbion.Node
}

func (G GRPCConnConfig) Clone() util.Param {
	return GRPCConnConfig{
		NewConn: G.NewConn,
		DstNode: proto.Clone(G.DstNode).(*pbion.Node),
	}
}

func (G GRPCConn) Lock(param util.Param, OnBroken func(badGay error)) error {
	G.conf = param.(GRPCConnConfig)
	conn, err := G.conf.NewConn(G.conf.DstNode.Service, G.conf.DstNode.Nid)
	if err != nil {
		return err
	}
	G.conn = conn
	G.onBroken = OnBroken
	return nil
}

func (G GRPCConn) OnBroken(err error) {
	if G.onBroken != nil {
		G.onBroken(err)
	}
}

func (G GRPCConn) Reconnect() {
	if G.onBroken != nil {
		G.onBroken(fmt.Errorf("GRPCConn should reconnect "))
	}
}

func (G GRPCConn) Repair(param util.Param, OnBroken func(badGay error)) error {
	if G.conn != nil {
		err := G.conn.Close()
		if err != nil {
			log.Warnf("error when G.conn.Close(): %+v", err)
		}
	}
	return G.Lock(param, OnBroken)
}

func (G GRPCConn) Update(param util.Param, OnBroken func(badGay error)) error {
	return G.Repair(param, OnBroken)
}

func (G GRPCConn) Remove() {
	if G.conn != nil {
		err := G.conn.Close()
		if err != nil {
			log.Warnf("error when G.conn.Close(): %+v", err)
		}
	}
	G.conn = nil
	G.onBroken = nil
}
