package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/smallnest/rpcx/client"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/kykira/kratos-rpcx-transport/internal/endpoint"
)

type discoveryResolver struct {
	w   registry.Watcher
	cc  *client.MultipleServersDiscovery
	log *log.Helper

	ctx    context.Context
	cancel context.CancelFunc

	insecure bool
}

func (r *discoveryResolver) watch() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		ins, err := r.w.Next()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			r.log.Errorf("[resolver] Failed to watch discovery endpoint: %v", err)
			time.Sleep(time.Second)
			continue
		}
		r.update(ins)
	}
}

func (r *discoveryResolver) update(ins []*registry.ServiceInstance) {
	var addrs []*client.KVPair
	for _, in := range ins {
		endpoint, err := endpoint.ParseEndpoint(in.Endpoints, "rpcx", r.insecure)
		if err != nil {
			r.log.Errorf("[resolver] Failed to parse discovery endpoint: %v", err)
			continue
		}
		if endpoint == "" {
			continue
		}
		value, _ := json.Marshal(in.Metadata)
		addr := &client.KVPair{
			Key:   endpoint,
			Value: string(value),
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		r.log.Warnf("[resolver] Zero endpoint found,refused to write, instances: %v", ins)
		return
	}
	r.cc.Update(addrs)
	b, _ := json.Marshal(ins)
	r.log.Infof("[resolver] update instances: %s", b)
}

func (r *discoveryResolver) Close() {
	r.cancel()
	err := r.w.Stop()
	if err != nil {
		r.log.Errorf("[resolver] failed to watch top: %s", err)
	}
}
