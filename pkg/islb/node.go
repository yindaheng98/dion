package islb

import (
	"context"
	"fmt"
	ndc "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/config"
	"google.golang.org/grpc"
	"sync"
)

//Node .
type Node struct {
	// Node ID
	NID string
	// Nats Client Conn
	nc *nats.Conn
	// gRPC Service Registrar
	nrpc *nrpc.Server
	// Service discovery client
	ndc *ndc.Client

	nodeLock sync.RWMutex
	//neighbor nodes
	neighborNodes map[string]discovery.Node

	cliLock sync.RWMutex
	clis    map[string]*nrpc.Client
}

//NewNode .
func NewNode(nid string) Node {
	return Node{
		NID:           nid,
		neighborNodes: make(map[string]discovery.Node),
		clis:          make(map[string]*nrpc.Client),
	}
}

//Start .
func (n *Node) Start(natURL string) error {
	var err error
	n.nc, err = util.NewNatsConn(natURL)
	if err != nil {
		log.Errorf("new nats conn error %v", err)
		n.Close()
		return err
	}
	n.ndc, err = ndc.NewClient(n.nc, config.DiscoveryLifeCycle)
	if err != nil {
		log.Errorf("new discovery client error %v", err)
		n.Close()
		return err
	}
	n.nrpc = nrpc.NewServer(n.nc, n.NID)
	return nil
}

//NatsConn .
func (n *Node) NatsConn() *nats.Conn {
	return n.nc
}

//KeepAlive Upload your node info to registry.
func (n *Node) KeepAlive(node discovery.Node) error {
	return n.ndc.KeepAlive(node)
}

func (n *Node) NewNatsRPCClientWithID(service, peerNID string, parameters map[string]interface{}) (*nrpc.Client, string, error) {
	var cli *nrpc.Client = nil
	selfNID := n.NID
	for id, node := range n.neighborNodes {
		if node.Service == service && (id == peerNID || peerNID == "*") {
			cli = nrpc.NewClient(n.nc, id, selfNID)
		}
	}

	if cli == nil {
		resp, err := n.ndc.Get(service, parameters)
		if err != nil {
			log.Errorf("failed to Get service [%v]: %v", service, err)
			return nil, "", err
		}

		if len(resp.Nodes) == 0 {
			err := fmt.Errorf("get service [%v], node cnt == 0", service)
			return nil, "", err
		}

		for _, node := range resp.Nodes {
			if node.NID == peerNID || peerNID == "*" {
				cli = nrpc.NewClient(n.nc, node.NID, selfNID)
			}
		}
	}

	if cli == nil {
		err := fmt.Errorf("get service [%v], but no %s", service, peerNID)
		return nil, "", err
	}

	n.cliLock.Lock()
	defer n.cliLock.Unlock()
	ID := util.RandomString(12)
	n.clis[ID] = cli
	return cli, ID, nil
}

func (n *Node) NewNatsRPCClient(service, peerNID string, parameters map[string]interface{}) (*nrpc.Client, error) {
	cli, _, err := n.NewNatsRPCClientWithID(service, peerNID, parameters)
	return cli, err
}

func (n *Node) NatsRPCClientByID(ID string) (*nrpc.Client, bool) {
	n.cliLock.Lock()
	defer n.cliLock.Unlock()
	cli, ok := n.clis[ID]
	return cli, ok
}

//Watch the neighbor nodes
func (n *Node) Watch(service string) error {
	resp, err := n.ndc.Get(service, map[string]interface{}{})
	if err != nil {
		log.Errorf("Watch service %v error %v", service, err)
		return err
	}

	for _, node := range resp.Nodes {
		n.handleNeighborNodes(discovery.NodeUp, &node)
	}

	return n.ndc.Watch(context.Background(), service, n.handleNeighborNodes)
}

// GetNeighborNodes get neighbor nodes.
func (n *Node) GetNeighborNodes() map[string]discovery.Node {
	n.nodeLock.Lock()
	defer n.nodeLock.Unlock()
	return n.neighborNodes
}

// handleNeighborNodes handle nodes up/down
func (n *Node) handleNeighborNodes(state discovery.NodeState, node *discovery.Node) {
	id := node.NID
	service := node.Service
	if state == discovery.NodeUp {
		log.Infof("Service up: "+service+" node id => [%v], rpc => %v", id, node.RPC.Protocol)
		if _, found := n.neighborNodes[id]; !found {
			n.nodeLock.Lock()
			n.neighborNodes[id] = *node
			n.nodeLock.Unlock()
		}
	} else if state == discovery.NodeDown {
		log.Infof("Service down: "+service+" node id => [%v]", id)

		n.nodeLock.Lock()
		delete(n.neighborNodes, id)
		n.nodeLock.Unlock()

		err := n.nrpc.CloseStream(id)
		if err != nil {
			log.Errorf("nrpc.CloseStream: err %v", err)
		}

		n.cliLock.Lock()
		defer n.cliLock.Unlock()
		for cid, cli := range n.clis {
			if cli.CloseStream(id) {
				delete(n.clis, cid)
			}
		}
	}
}

//ServiceRegistrar return grpc.ServiceRegistrar of this node, used to create grpc services
func (n *Node) ServiceRegistrar() grpc.ServiceRegistrar {
	return n.nrpc
}

//Close .
func (n *Node) Close() {
	if n.nrpc != nil {
		n.nrpc.Stop()
	}
	if n.nc != nil {
		n.nc.Close()
	}
	if n.ndc != nil {
		n.ndc.Close()
	}
}
