package helpers

import (
	"errors"
	"fmt"
	"net"
)

func GetAvailablePort(startPort, endPort int) (int, error) {
	for port := startPort; port <= endPort; port++ {
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, errors.New("Unable to find free port")
}
