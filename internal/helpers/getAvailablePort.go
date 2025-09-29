package helpers

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/substrate-cli/consumer-service-cli/internal/utils"
)

func GetAvailablePort(startPort, endPort int) (int, error) {
	if utils.GetBundle() == "dockerrerr" {
		// find free port on host
		ln, _ := net.Listen("tcp", ":0")
		hostPort := ln.Addr().(*net.TCPAddr).Port
		ln.Close()

		// build docker run command
		runCmd := exec.Command("docker", "run",
			"-d", "--rm",
			"--name", "generated-app",
			"--network", "appnet",
			"-p", fmt.Sprintf("%d:3000", hostPort),
			"generated-app:latest",
		)

		// run docker container
		output, err := runCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Output:", string(output))
			return 0, err
		}

		// container ID returned by docker
		containerID := string(output)
		fmt.Printf("Container startedddddddddddd333333333333333: %s\n", containerID)
		fmt.Printf("App is running on http://localhost:%d\n", hostPort)

		return hostPort, nil
	}
	for port := startPort; port <= endPort; port++ {
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close() // Port is available, close it immediately
			return port, nil
		}
	}
	return 0, errors.New("Unable to find free port")
}
