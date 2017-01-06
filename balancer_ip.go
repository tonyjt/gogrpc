package gogrpc

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"math/rand"
	"sync"
)

type BalancerIp struct {
	chanNotify chan []grpc.Address
	addr       []grpc.Address
	addrUp     []grpc.Address
	addrDown   []grpc.Address
	mu         sync.Mutex
}

func NewBalancerIp() *BalancerIp {
	balancer := &BalancerIp{}
	balancer.chanNotify = make(chan []grpc.Address)
	return balancer
}

func (b *BalancerIp) SetAddr(ip ...string) error {

	if len(ip) > 0 {
		var addrs []grpc.Address
		for _, p := range ip {
			a := grpc.Address{Addr: p}
			addrs = append(addrs, a)
		}
		go func() {
			b.chanNotify <- addrs
		}()
		b.addr = append(b.addr, addrs...)
		return nil
	}
	return errors.New("ips is empty")
}

// Start does the initialization work to bootstrap a Balancer. For example,
// this function may start the name resolution and watch the updates. It will
// be called when dialing.
func (b *BalancerIp) Start(target string, config grpc.BalancerConfig) error {
	return nil
}

// Up informs the Balancer that gRPC has a connection to the server at
// addr. It returns down which is called once the connection to addr gets
// lost or closed.
// TODO: It is not clear how to construct and take advantage of the meaningful error
// parameter for down. Need realistic demands to guide.
func (b *BalancerIp) Up(addr grpc.Address) (down func(error)) {

	b.mu.Lock()
	for i, a := range b.addrDown {
		if addr.Addr == a.Addr {
			b.addrDown = append(b.addrDown[:i], b.addrDown[i+1:]...)
			break
		}
	}
	for i := 0; i < len(b.addrUp); i++ {
		if b.addrUp[i].Addr == addr.Addr {
			break
		}

		if i == len(b.addrUp)-1 {
			b.addrUp = append(b.addrUp, addr)
			break
		}
	}

	b.mu.Unlock()
	return func(error) {
		b.mu.Lock()
		for i, a := range b.addrUp {
			if addr.Addr == a.Addr {
				b.addrUp = append(b.addrUp[:i], b.addrUp[i+1:]...)
				break
			}
		}
		for i := 0; i < len(b.addrDown); i++ {
			if b.addrDown[i].Addr == addr.Addr {
				break
			}

			if i == len(b.addrDown)-1 {
				b.addrUp = append(b.addrDown, addr)
				break
			}
		}

		b.mu.Unlock()
	}
}

func (b *BalancerIp) Get(ctx context.Context, opts grpc.BalancerGetOptions) (addr grpc.Address, put func(), err error) {
	if len(b.addrUp) == 0 {
		err = errors.New("all addr is down")

	} else if len(b.addrUp) == 1 {
		addr = b.addrUp[0]

	} else {
		index := rand.Intn(len(b.addrUp))
		put = nil
		addr = b.addrUp[index]

	}
	return
}

func (b *BalancerIp) Notify() <-chan []grpc.Address {
	return b.chanNotify
}

func (b *BalancerIp) Close() error {
	close(b.chanNotify)
	b.addr = nil
	b.addrUp = nil
	b.addrDown = nil
	return nil
}
