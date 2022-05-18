package sxu

import (
	"github.com/pion/ion/pkg/ion"
	"google.golang.org/grpc"
	"sync"
)

type connID string
type peerConnList map[string]connID
type serviceList map[string]peerConnList

type NRPCConnPool struct {
	node       *ion.Node
	Parameters map[string]interface{}
	connList   serviceList
	sync.Mutex
}

func NewNRPCConnPool(node *ion.Node) *NRPCConnPool {
	return &NRPCConnPool{
		node:     node,
		connList: serviceList{},
	}
}

func (N *NRPCConnPool) GetConn(service, peerNID string) (grpc.ClientConnInterface, error) {
	N.Lock()
	defer N.Unlock()
	if pcl, ok := N.connList[service]; ok { // peerConnList exist in serviceList?
		if cid, ok := pcl[peerNID]; ok { // Conn exist in peerConnList?
			if conn, ok := N.node.NatsRPCClientByID(string(cid)); ok { // Conn exist in node?
				return conn, nil // return it
			}
			// Conn not exist in node? should recreate it
		} // Conn not exist in peerConnList? should create it
		// Anyway, should create it
		conn, cid, err := N.node.NewNatsRPCClientWithID(service, peerNID, N.Parameters)
		if err != nil {
			return nil, err
		}
		pcl[peerNID] = connID(cid)
		N.connList[service] = pcl
		return conn, nil
	} // peerConnList not exist in serviceList? should create it
	pcl := map[string]connID{}
	conn, cid, err := N.node.NewNatsRPCClientWithID(service, peerNID, N.Parameters)
	if err != nil {
		return nil, err
	}
	pcl[peerNID] = connID(cid)
	N.connList[service] = pcl
	return conn, nil
}
