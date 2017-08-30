package gogrpc

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
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

	l, err := server.net.Listen(server.nett, server.laddr)
	if err != nil {
		return err
	}
	server.listener = l

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
		if err := server.gs.Serve(server.listener); err != nil {
			glog.Infof("serve error:%s \n", err.Error())
		}
	}()

	// Close the parent if we inherited and it wasn't init that started us.
	if didInherit && ppid != 1 {
		glog.Infof("Time to kill my parent, ppid: %d", ppid)
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to close parent - %d: %s", ppid, err)
		}
	}

	server.signalHandler()

	if *verbose {
		glog.Infof("Exiting pid %d.", os.Getpid())
	}
	return nil
}

//GetServer get grpc server
func (server *Server) GetServer() *grpc.Server {
	return server.gs
}

func (server *Server) signalHandler() {
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGUSR2:
			// on SIGUSR2, start a new process, then wait to be killed.
			if pid, err := server.net.StartProcess(); err != nil {
				glog.Infof("start new process failed,err:%s \n", err.Error())
			} else {
				glog.Infof("start new process, pid:%d \n", pid)
			}
		default:
			// stop on other cases, since you are a parent being killed
			wg := sync.WaitGroup{}
			wg.Add(2)
			go func() {
				server.gs.GracefulStop()
				wg.Done()
			}()
			go func() {
				server.Close()
				wg.Done()
			}()
			wg.Wait()
			return
		}
	}
}
