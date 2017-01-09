package gogrpc

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	//"log"
	"math/rand"
	"sync"
	"time"
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
	//log.Printf("up :addr:%s\n", addr.Addr)
	//b.mu.Lock()
	for i, a := range b.addrDown {
		if addr.Addr == a.Addr {
			b.addrDown = append(b.addrDown[:i], b.addrDown[i+1:]...)
			break
		}
	}

	for i, a := range b.addr {
		if addr.Addr == a.Addr {
			b.addr = append(b.addr[:i], b.addr[i+1:]...)
			break
		}
	}
	var iUp int
	for iUp = 0; iUp < len(b.addrUp); iUp++ {
		if b.addrUp[iUp].Addr == addr.Addr {
			break
		}
	}
	if iUp >= len(b.addrUp) {
		b.addrUp = append(b.addrUp, addr)
	}
	address := addr.Addr
	//b.mu.Unlock()
	down = func(err error) {
		//log.Printf("down :addr:%s,err:%s \n", address, err.Error())
		//b.mu.Lock()
		for i, a := range b.addrUp {
			if address == a.Addr {
				b.addrUp = append(b.addrUp[:i], b.addrUp[i+1:]...)
				break
			}
		}

		var iDown int
		for iDown = 0; iDown < len(b.addrDown); iDown++ {
			if b.addrDown[iDown].Addr == address {
				break
			}
		}
		if iDown >= len(b.addrDown) {
			b.addrDown = append(b.addrDown, addr)
		}
		//b.mu.Unlock()
	}
	return
}

func (b *BalancerIp) Get(ctx context.Context, opts grpc.BalancerGetOptions) (addr grpc.Address, put func(), err error) {

	//log.Println("Get")
	if len(b.addr) == 0 && len(b.addrUp) == 0 {
		err = errors.New("all addr is down")
	} else {
		rand.Seed(time.Now().UTC().UnixNano())

		var index int

		if len(b.addr) > 0 {
			index = rand.Intn(len(b.addr))
			//log.Printf("len addr: %d,index :%d\n", len(b.addr), index)
			addr = b.addr[index]

		} else {
			index = rand.Intn(len(b.addrUp))
			//log.Printf("len up :%d,index :%d\n", len(b.addrUp), index)
			addr = b.addrUp[index]
		}

		put = nil

	}
	return
}

func (b *BalancerIp) Notify() <-chan []grpc.Address {
	return b.chanNotify
}

func (b *BalancerIp) Close() error {
	close(b.chanNotify)
	b.addr = nil
	b.addrDown = nil
	b.addrUp = nil
	return nil
}
