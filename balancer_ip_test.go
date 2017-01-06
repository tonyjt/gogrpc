package gogrpc

import (
	"testing"

	"google.golang.org/grpc"
)

func TestBalancerIp(t *testing.T) {

	b := NewBalancerIp()

	err := b.SetAddr("172.172.177.19:7001", "172.172.177.34:7001")

	var c *grpc.ClientConn
	c, err = grpc.Dial("123", grpc.WithBalancer(b), grpc.WithInsecure())

	if err != nil {
		t.Errorf("dial failed:%s", err.Error())
	} else {
		c.Close()
	}

}
