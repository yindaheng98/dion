package room

import (
	"context"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

type Service struct {
	pb.UnimplementedRoomServer
	set     *util.ExpireSetMap[string, string]
	eventCh chan *syncer.SessionEvent
}

func NewService() Service {
	set := util.NewExpireSetMap[string, string](config.ClientSessionExpire)
	eventCh := make(chan *syncer.SessionEvent, 64)
	set.OnDelete(func(user string, session string) {
		eventCh <- &syncer.SessionEvent{
			Session: &pb.ClientNeededSession{
				Session: session,
				User:    user,
			},
			State: syncer.SessionEvent_REMOVE,
		}
	})
	return Service{
		set:     set,
		eventCh: eventCh,
	}
}

func (r Service) ClientHealth(_ context.Context, session *pb.ClientNeededSession) (*pb.HealthReply, error) {
	r.set.Start()
	r.set.Update(session.User, session.Session)
	r.eventCh <- &syncer.SessionEvent{
		Session: session,
		State:   syncer.SessionEvent_ADD,
	}
	return &pb.HealthReply{Ok: true}, nil
}

func (r Service) FetchSessionEvent() *syncer.SessionEvent {
	return <-r.eventCh
}

// 为何不在sfu中实现而要单走？因为连客户端是SXU独有的功能，而sfu里的信令服务是大家都有的
