package gogrpc

import (
	"net"
	"testing"

	"github.com/facebookgo/grace/gracenet"
)

func testInitNewServer() *Server {

	return NewServer("tcp", "localhost:7001")
}

func TestServer_Serve(t *testing.T) {
	s := testInitNewServer()

	go s.Serve()
}

func TestListenClose(t *testing.T) {
	nett := &gracenet.Net{}
	//var l net.TCPListener
	l, err := nett.Listen("tcp", "localhost:9001")

	if err != nil {
		t.Errorf("open 1 :%s", err.Error())
	}
	f, errf := l.(*net.TCPListener).File()

	if errf != nil {
		t.Errorf("errf :%s", errf.Error())
	}

	f.Close()

	_, err = nett.Listen("tcp", "localhost:9001")

	if err != nil {
		t.Errorf("open 3 :%s", err.Error())
	}

}
