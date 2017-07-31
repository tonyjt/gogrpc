package gogrpc

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/facebookgo/grace/gracenet"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

var (
	verbose    = flag.Bool("gracegrpc.log", true, "Enable logging.")
	didInherit = os.Getenv("LISTEN_FDS") != ""
	ppid       = os.Getppid()
)

type ServeCloser interface {
	Close()
}

// An app contains one or more servers and associated configuration.
type Server struct {
	net      *gracenet.Net
	gs       *grpc.Server
	listener net.Listener
	errors   chan error
	nett     string
	laddr    string
	ServeCloser
}

// NewServer new server, nett: net, laddr: address ,opt grpc server options
func NewServer(nett, laddr string, opt ...grpc.ServerOption) *Server {

	server := &Server{
		net:   &gracenet.Net{},
		gs:    grpc.NewServer(opt...),
		nett:  nett,
		laddr: laddr}

	return server
}

//Serve serve
func (server *Server) Serve() error {

	err := server.listen()
	if err != nil {
		return err
	}

	// Some useful logging.
	if *verbose {
		if didInherit {
			if ppid == 1 {
				glog.Infof("Listening on init activated %s", server.listener.Addr())
			} else {
				const msg = "Graceful handoff of %s with new pid %d and old pid %d"
				glog.Infof(msg, server.listener.Addr(), os.Getpid(), ppid)
			}
		} else {
			const msg = "Original serving %s with pid %d parent: %d"
			glog.Infof(msg, server.listener.Addr(), os.Getpid(), ppid)
		}
	}
	//

	go func() {
		// Start serving
		if err := server.serve(); err != nil {
			glog.Infof("serve error:%s \n", err.Error())
		}
	}()

	server.signalHandler()

	// Close the parent if we inherited and it wasn't init that started us.
	if didInherit && ppid != 1 {
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to close parent: %s", err)
		}
	}

	if *verbose {
		glog.Infof("Exiting pid %d.", os.Getpid())
	}
	return nil
}

//GetServer get grpc server
func (server *Server) GetServer() *grpc.Server {
	return server.gs
}

func (server *Server) listen() error {
	l, err := server.net.Listen(server.nett, server.laddr)

	if err != nil {
		return err
	}
	server.listener = l
	return nil
}

func (server *Server) serve() error {

	return server.gs.Serve(server.listener)
}

func (server *Server) signalHandler() {
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP:
			// this ensures a subsequent INT/TERM will trigger standard go behaviour of
			// terminating.
			// signal.Stop(ch)

			server.gs.GracefulStop()
			server.Close()
			return
		case syscall.SIGUSR2:
			// we only return here if there's an error, otherwise the new process
			// will send us a TERM when it's ready to trigger the actual shutdown.

			if pid, err := server.net.StartProcess(); err != nil {
				glog.Infof("start new process failed,err:%s \n", err.Error())
			} else {
				glog.Infof("start new process, pid:%d \n", pid)
			}
			server.gs.GracefulStop()
			server.Close()
			return
			//server.gs.Stop()
		}
	}
}
