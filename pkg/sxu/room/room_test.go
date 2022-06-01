package room

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/yindaheng98/dion/algorithms/impl/random"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"testing"
	"time"
)

var conf = config.Common{
	Global: config.Global{Dc: "dc1"},
	Log:    config.LogConf{Level: "DEBUG"},
	Nats:   config.NatsConf{URL: "nats://192.168.1.2:4222"},
}

func testRoomService(t *testing.T) {
	node := ion.NewNode("room-test-" + util.RandomString(4))
	err := node.Start(conf.Nats.URL)
	if err != nil {
		t.Error(err)
	}
	s := NewService()
	go func() {
		for {
			t.Logf("SessionEvent: %+v", s.FetchSessionEvent())
		}
	}()
	//grpc service
	pb.RegisterRoomServer(node.ServiceRegistrar(), s)

	// Register reflection service on nats-rpc server.
	reflection.Register(node.ServiceRegistrar().(*nrpc.Server))

	//重要！！！必须开启了Watch才能自动地关闭NATS GRPC连接.
	go func() {
		err := node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	//重要！！！必须开启了KeepAlive才能在退出时让服务端那边自动地关闭NATS GRPC连接.
	go func() {
		err := node.KeepAlive(discovery.Node{
			DC:      conf.Global.Dc,
			Service: config.ServiceSXU,
			NID:     node.NID,
			RPC: discovery.RPC{
				Protocol: discovery.NGRPC,
				Addr:     conf.Nats.URL,
				//Params:   map[string]string{"username": "foo", "password": "bar"},
			},
		})
		if err != nil {
			log.Errorf("isglb.Node.KeepAlive(%v) error %v", node.NID, err)
		}
	}()
}
func TestRoomService(t *testing.T) {
	for i := 0; i < 10; i++ {
		go testRoomService(t)
	}
	select {}
}

func testRoomClient(t *testing.T) {
	node := ion.NewNode("room-cli")
	err := node.Start(conf.Nats.URL)
	if err != nil {
		t.Error(err)
	}
	//重要！！！必须开启了Watch才能获取到其他节点.
	go func() {
		err := node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	c := NewClient(&node, random.RandomSelector{}, map[string]interface{}{})
	c.UpdateSession(&pb.ClientNeededSession{Session: util.RandomString(4), User: util.RandomString(4)})
	c.Connect()
	for i := 0; i < 100; i++ {
		<-time.After(1 * time.Second)
		if random.RandBool() {
			c.UpdateSession(&pb.ClientNeededSession{Session: util.RandomString(4), User: util.RandomString(4)})
		}
		if random.RandBool() {
			c.RefreshConn()
		}
	}
}

func TestRoomClient(t *testing.T) {
	for i := 0; i < 10; i++ {
		go testRoomClient(t)
	}
	select {}
}
