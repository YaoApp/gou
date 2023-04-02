package http

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/log"
)

// New create a new http server
func New(router *gin.Engine, option Option) *Server {

	if option.Timeout == 0 {
		option.Timeout = 5 * time.Second
	}

	return &Server{
		router: router,
		option: &option,
		signal: make(chan uint8, 1),
		event:  make(chan uint8, 1),
		status: CREATED,
	}
}

// Event get event signal
func (server *Server) Event() chan uint8 {
	return server.event
}

// Port get server port
func (server *Server) Port() (int, error) {
	addr := strings.Split(server.addr.String(), ":")
	if len(addr) != 2 {
		return 0, fmt.Errorf("can't get port %s", server.addr.String())
	}
	port, err := strconv.Atoi(addr[1])
	if err != nil {
		return 0, err
	}
	return port, nil
}

// Ready check if the status is ready
func (server *Server) Ready() bool {
	return server.status == READY
}

// Start a http server
func (server *Server) Start() error {

	switch server.status {
	case READY:
		return fmt.Errorf("server already started")

	case STARTING:
		return fmt.Errorf("server is starting")
	}

	server.status = STARTING

	// Server Setting
	var listener net.Listener
	var err error
	addr := fmt.Sprintf("%s:%d", server.option.Host, server.option.Port)
	listener, err = net.Listen("tcp4", addr)
	if err != nil {
		log.Error("[Server] %s %s", addr, err.Error())
		server.status = CREATED
		server.event <- ERROR
		return err
	}

	// network preparing
	server.addr = listener.Addr()
	srv := &http.Server{Addr: server.addr.String(), Handler: server.router}

	// close server
	defer func() {
		if server.status == RESTARTING {
			return
		}

		log.Info("[Server] %s was closed", srv.Addr)
		err := srv.Close()
		if err != nil {
			log.Error("[Server] %s %s", srv.Addr, err.Error())
		}

		server.status = CLOSED
		server.event <- CLOSE
	}()

	// WebSocket
	if len(websocket.Upgraders) > 0 {

		// Start WebSocket Hub
		for id, upgrader := range websocket.Upgraders {
			upgrader.SetRouter(server.router)
			go upgrader.Start()
			log.Info("Websocket %s start", id)
		}

		// Stop WebSocket Hub
		defer func() {
			for id, upgrader := range websocket.Upgraders {
				upgrader.Stop()
				log.Info("Websocket %s quit", id)
			}
		}()
	}

	// start server
	go func() {
		server.status = READY
		server.event <- READY
		if errSrv := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			err = errSrv
			log.Error("[Server] %s %s", srv.Addr, errSrv.Error())
			server.signal <- ERROR
		}
	}()

	// make a timer
	timer := time.NewTimer(time.Duration(server.option.Timeout))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if server.Ready() {
				timer.Stop()
				break
			}

			log.Error("[Server] %s start operation was canceled (timeout %d)", srv.Addr, server.option.Timeout)
			return fmt.Errorf("canceled (timeout %d)", server.option.Timeout)

		case signal := <-server.signal:
			switch signal {

			case READY:
				log.Info("[Server] %s is ready", srv.Addr)
				break

			case CLOSE:
				err = listener.Close()
				if err != nil {
					log.Error("[Server] %s close error (%s)", srv.Addr, err.Error())
					return err
				}

				err = srv.Close()
				if err != nil {
					log.Error("[Server] %s restarting (%s)", srv.Addr, err.Error())
					return err
				}

				log.Info("[Server] %s was closed", srv.Addr)
				return nil

			case RESTART:
				log.Info("[Server] %s was closed (for restarting)", srv.Addr)
				server.status = RESTARTING

				err = listener.Close()
				if err != nil {
					log.Error("[Server] %s restarting (%s)", srv.Addr, err.Error())
					return err
				}

				err = srv.Close()
				if err != nil {
					log.Error("[Server] %s restarting (%s)", srv.Addr, err.Error())
					return err
				}

				defer server.Start()
				return nil

			case ERROR:
				log.Error("[Server] %s was closed (%s)", srv.Addr, err.Error())
				return err

			default:
				log.Error("[Server] %s was closed (unknown signal %d)", srv.Addr, signal)
				return fmt.Errorf("get an unknown signal %d", signal)
			}
		}
	}
}

// Stop a http server
func (server *Server) Stop() error {
	if server.status != READY {
		return fmt.Errorf("server is not ready")
	}
	server.signal <- CLOSE
	return nil
}

// Restart a http server
func (server *Server) Restart() error {
	if server.status != READY {
		return fmt.Errorf("server is not ready")
	}
	server.signal <- RESTART
	return nil
}

// With middlewares
func (server *Server) With(middlewares ...func(ctx *gin.Context)) *Server {
	for _, middleware := range middlewares {
		server.router.Use(middleware)
	}
	return server
}
