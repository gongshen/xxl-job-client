package executor

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
)

type Executor struct {
	AppName    string
	Port       int
	httpServer *fasthttp.Server
	cancel     func() error
}

func NewExecutor(appName string, port int) *Executor {
	return &Executor{
		AppName: appName,
		Port:    port,
	}
}
func (e *Executor) SetServer(srv *fasthttp.Server) {
	e.httpServer = srv
}

func (e *Executor) GetRegisterAddr() string {
	return fmt.Sprintf("http://%s:%d/", getIp(), e.Port)
}

func getLocalIP() string {
	ip := getIPFromInterface("eth0")
	if ip == "" {
		ip = getIPFromInterface("en0")
	}
	if ip == "" {
		panic("Unable to determine local IP address (non loopback). Exiting.")
	}
	return ip
}

func getIPFromInterface(interfaceName string) string {
	itf, _ := net.InterfaceByName(interfaceName)
	item, _ := itf.Addrs()
	var ip net.IP
	for _, addr := range item {
		switch v := addr.(type) {
		case *net.IPNet:
			if !v.IP.IsLoopback() {
				if v.IP.To4() != nil {
					ip = v.IP
				}
			}
		}
	}
	if ip != nil {
		return ip.String()
	} else {
		return ""
	}
}

func getIp() string {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	panic("Unable to determine local IP address (non loopback). Exiting.")
}

func (e *Executor) Run() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", e.Port))
	if err != nil {
		panic(err)
	}
	e.cancel = func() error {
		return ln.Close()
	}
	return e.httpServer.Serve(ln)
}

func (e *Executor) Close() error {
	return e.cancel()
}
