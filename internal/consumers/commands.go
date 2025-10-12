package consumers

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/substrate-cli/consumer-service-cli/internal/helpers"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
	"github.com/substrate-cli/consumer-service-cli/internal/webhooks"
)

func createNextApp(projectPath string) error {
	// basePath := filepath.Join(projectPath)
	// Create Next.js app using npx (must be installed globally)
	cmd := exec.Command(
		"npx",
		"--yes",
		"create-next-app@latest",
		"app",
		"--ts",
		"--eslint",
		"--no-tailwind",
		"--src-dir",
		"--app",
		"--import-alias", "@/*",
		"--no-turbopack",
	)
	cmd.Dir = projectPath // important: run the command from ~/Desktop
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Println(err)
		return err
	}
	projectPath = filepath.Join(projectPath, "app")
	cmdTailwind := exec.Command("npm", "install", "-D", "tailwindcss@3", "postcss", "autoprefixer")

	cmdTailwind.Dir = projectPath // same as cwd
	cmdTailwind.Stdout = os.Stdout
	cmdTailwind.Stderr = os.Stderr
	cmdTailwind.Stdin = os.Stdin // optional, for prompts

	if err := cmdTailwind.Run(); err != nil {
		return err
	}

	cmdInitTailwind := exec.Command("npx", "tailwindcss", "init", "-p")

	cmdInitTailwind.Dir = projectPath // set working directory like cwd
	cmdInitTailwind.Stdout = os.Stdout
	cmdInitTailwind.Stderr = os.Stderr
	cmdInitTailwind.Stdin = os.Stdin // optional

	if err := cmdInitTailwind.Run(); err != nil {
		return err
	}

	log.Print("Tailwind config installed successfully.")
	// writing tailwind config -----
	configPath := filepath.Join(projectPath, "tailwind.config.js")
	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Println("failed to read config: %w", err)
		return err
	}

	content := string(contentBytes)
	old := "content: [],"
	new := `content: ["./src/**/*.{js,ts,jsx,tsx}", "./pages/**/*.{js,ts,jsx,tsx}", "./components/**/*.{js,ts,jsx,tsx}"],`
	updated := strings.Replace(content, old, new, 1)

	err = os.WriteFile(configPath, []byte(updated), 0644)
	if err != nil {
		return err
	}
	log.Print("Tailwind config installed and configured successfully.")

	globalsCssPath := filepath.Join(projectPath, "src", "app", "globals.css")
	tailwindImports := "@tailwind base;\n@tailwind components;\n@tailwind utilities;\n\n"
	existing, err := os.ReadFile(globalsCssPath)
	if err != nil {
		return err
	}

	newContent := []byte(tailwindImports + string(existing))
	err = os.WriteFile(globalsCssPath, newContent, 0644)
	if err != nil {
		return err
	}
	log.Print("Tailwind and globals.css configured successfully.")

	return nil
}

func createNodeJSProject(projectPath string) error {
	serverPath := filepath.Join(projectPath, "server")

	cmd := exec.Command("npm", "init", "-y")
	cmd.Dir = serverPath // important: run the command from ~/Desktop
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create next app: %w", err)
	}

	cmd = exec.Command("npm", "install", "express", "cors", "dotenv")

	// Set the working directory
	cmd.Dir = serverPath

	// Pipe stdout, stderr, and stdin to the parent terminal (like stdio: 'inherit')
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create server app: %w", err)
	}

	return nil
}

func generateCode(appPath string, data map[string]interface{}, structure string, cluster string, initCommands [][]string) error {
	rawMap, ok := data["fileStructure"].(map[string]interface{})
	rawLibraries, ok2 := data["libraries"].([]interface{})
	rawCommands, ok3 := data["additionalCommands"].([]interface{})
	var libraries []string
	var commands []string
	if !ok2 {
		log.Println("libraries key not found or not an array")
	} else {

		for _, lib := range rawLibraries {
			if str, ok := lib.(string); ok {
				libraries = append(libraries, str)
			}
		}
		log.Println("Parsed libraries:", libraries)
	}

	if !ok {
		log.Println("fileStructure is not a map[string]interface{}")
		errW := webhooks.ErrorAction("finished", "error creating cluster", "invalid file structure")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return errors.New("invalid file structure")
	}

	if !ok3 {
		log.Println("additionalCommands key not found or not an array")
	} else {
		if len(rawCommands) > 0 {
			for _, lib := range rawCommands {
				if str, ok := lib.(string); ok {
					commands = append(commands, str)
				}
			}
			log.Println("Parsed commands:", commands)
		}
	}

	fileMap := make(map[string]string)

	if utils.GetBundle() == "docker" {
		log.Println("docker environment detected...")
		log.Println("generating docker file for => ", structure)
		stm := fmt.Sprintf("Generating dockerfile for %s", structure)
		errW := webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		dockerfile := helpers.GetDockerFile(structure)
		rawMap["dockerfile"] = map[string]interface{}{
			"code": dockerfile,
		} //only for next js for now -----
	}

	for path, content := range rawMap {

		if m, ok := content.(map[string]interface{}); ok {
			if content, exists := m["code"]; exists {
				strContent, ok := content.(string)
				if !ok {
					smt := fmt.Sprintf("content for path %s is not a string", path)
					log.Println(smt)
				}
				fileMap[path] = strContent
			}
		} else {
			fmt.Printf("Path %s is not a map, got %T\n", path, content)
		}

	}

	errW := webhooks.PrecheckAction("finished", "installing libraries... DO NOT QUIT")
	if errW != nil {
		log.Println("api-service webhook failed")
	}

	err := installLibraries(appPath, libraries)
	if err != nil {
		log.Println("Failed to install libraries")
		errW := webhooks.ErrorAction("finished", "Failed to install libraries", err.Error())
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}

	if (len(commands)) > 0 {

		errW = webhooks.PrecheckAction("finished", "running additional commands... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		err = runAdditionalCommands(appPath, commands, initCommands)
		if err != nil {
			log.Println("command execution failed")
			errW := webhooks.ErrorAction("finished", "command execution failed", err.Error())
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
	}

	errW = webhooks.PrecheckAction("finished", "Creating Directories... DO NOT QUIT")
	if errW != nil {
		log.Println("api-service webhook failed")
	}

	// log.Println("fileMap ready:", fileMap)
	err = utils.CreateDirectories(appPath, fileMap)
	if err != nil {
		log.Print("Unable to generate project.", err)
		return err
	}

	// rawMap, ok = data["fileStructure"].(map[string]interface{})
	// if !ok {
	// 	log.Fatal("fileCodes is not a map[string]interface{}")
	// }

	fileContentMapRedis := make(map[string]string)
	for path, content := range rawMap {
		// strContent, ok := content.(string)
		if m, ok := content.(map[string]interface{}); ok {
			if code, exists := m["code"]; exists {
				relPath := filepath.Join(appPath, path)
				err := utils.WriteFiles(relPath, code.(string))
				if err != nil {
					log.Println("error writing files")
					return err
				}
				fileContentMapRedis[path] = code.(string)
			}

			// jsonData, err := json.Marshal(fileContentMapRedis)
			// if err != nil {
			// 	log.Println("Error marshaling map:", err)
			// 	return err
			// }

			// //saving to redis ----
			// stm := fmt.Sprintf("%s:structure", cluster)
			// db.SaveRedis(stm, string(jsonData))
		} else {
			fmt.Printf("Path %s is not a map, got %T\n", path, content)
		}
	}
	return nil
}

func installLibraries(rootPath string, libraries []string) error {
	libArgs := append([]string{"install"}, libraries...)
	// Build the full path for cwd
	cwd := filepath.Join(rootPath)

	// Create the command
	cmd := exec.Command("npm", libArgs...)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Run the command
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to install some dependencies in %s: %v\n", cwd, err)
		// Continue execution even if install fails, just like the JS `resolve()`
		return nil
	}

	log.Println("Dependencies installed successfully in", cwd)
	return nil
}

func runAdditionalCommands(rootPath string, commands []string, initCommands [][]string) error {
	libArgs := commands
	cwd := filepath.Join(rootPath)
	isError := false

	for index, args := range initCommands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if index == 0 {
				isError = true
			}
			log.Printf("Failed init command: %v\n", args)
		}
	}

	// Create the command
	if !isError {
		for _, cmdStr := range libArgs {
			parts := strings.Split(cmdStr, " ")
			cmd := exec.Command("npx", parts...) // pass as separate arrgs ----

			cmd.Dir = cwd

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				log.Printf("Failed to install some dependencies in %s: %v\n", cwd, err)
				// Continue execution even if install fails, just like the JS `resolve()`
				return nil
			}
		}
	}

	log.Println("Dependencies installed successfully in", cwd)
	return nil
}

func runProject(rootPath string, isbackend bool, cluster string, isFS bool) error {
	type error interface {
		Error() string
	}
	var err error
	if isbackend {

		err = runServer(rootPath, cluster, isFS)
		if err != nil {
			log.Println("Failed to start server")
			log.Println(err)
			return err
		}
	}
	err = runApp(rootPath, cluster, isFS)
	if err != nil {
		log.Println("Failed to start app")
		log.Println(err)
		return err
	}
	return nil
}

func runServer(rootPath string, cluster string, isFS bool) error {
	path := filepath.Join(rootPath, "server")
	cmd := exec.Command("npm", "run", "dev")
	// port, err := helpers.GetAvailablePort(3000, 3100)
	// if err != nil {
	// 	log.Println("Terminating execution, no free port available.")
	// 	return err
	// }
	if utils.GetBundle() == "docker" && isFS {
		log.Println("docker environment detected for server...")
		return nil
	}

	cmd.Dir = path // same as cwd
	log.Println("server will start on PORT =>", backendPort)
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(backendPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // optional, for prompts

	// Start the process without waiting for it to finish
	if err := cmd.Start(); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)
	return nil
}

func runApp(rootPath string, cluster string, isFS bool) error {
	path := filepath.Join(rootPath, "app")
	cmd := exec.Command("npm", "run", "dev")
	log.Println("starting next js server...")
	// if err != nil {
	// 	log.Println("Terminating execution, no free port available.")
	// 	return err
	// }
	if utils.GetBundle() == "docker" && !isFS {
		log.Println("docker environment detected for cluster...")
		imageName := fmt.Sprintf("%s:latest", cluster)
		containerName := cluster
		buildCmd := exec.Command("docker", "build", "-t", imageName, path)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		stm := "building docker image for app... DO NOT QUIT"
		errW := webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("running docker build...")
		if err := buildCmd.Run(); err != nil {
			fmt.Println("❌ Error building image:", err)
			return err
		}

		stm = "docker image build successful... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		//gettign post from host machine -----
		stm = "searching for free port on host machine... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("finding free port on host machine...")
		ln, _ := net.Listen("tcp", ":0")
		hostPort := ln.Addr().(*net.TCPAddr).Port
		appPort = hostPort
		ln.Close()

		stm = "Warming container in docker daemon... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("starting container...")
		runCmd := exec.Command("docker", "run",
			"-d", "--rm",
			"--name", containerName,
			"-p", fmt.Sprintf("%d:3000", hostPort),
			imageName,
		)

		// run docker container
		output, err := runCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Output:", string(output))
			return err
		}

		log.Println("container running...")

		stm = "cluster running in container... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		// container ID returned by docker
		containerID := string(output)
		fmt.Printf("Container started: %s\n", containerID)
		fmt.Printf("App is running on http://localhost:%d\n", hostPort)

		stm = "cluster running on " + strconv.Itoa(hostPort)
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		fmt.Println("✅ Generated app container started")
		path = filepath.Join(path, "node_modules")
		utils.DeleteFile(path)
		return nil
	}

	if utils.GetBundle() == "docker" && isFS {
		//running docker-compose ----
		log.Println("docker environment detected for cluster...")
		stm := "finding free ports on host machine... DO NOT QUIT"
		errW := webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("searching for free ports on host machine...")
		ln1, _ := net.Listen("tcp", ":0")
		backendPort = ln1.Addr().(*net.TCPAddr).Port

		ln2, _ := net.Listen("tcp", ":0")
		appPort = ln2.Addr().(*net.TCPAddr).Port

		ln1.Close()
		ln2.Close()

		stm = "Writing Docker Compose to warm up cluster... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("writing docker compose...")
		compose := helpers.GetDockerCompose(backendPort, appPort, cluster)
		composePath := filepath.Join(rootPath, "docker-compose.yml")
		_ = utils.WriteFiles(composePath, compose)

		stm = "Docker compose acknowledged, building images... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("docker compose written successfully...")
		log.Println("building docker images...")
		buildCmd := exec.Command("docker", "compose", "build")
		buildCmd.Dir = rootPath
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			log.Println("❌ Error building image:", err)
			return err
		}

		log.Println("images built successfully...")
		stm = "build successful, warming up containers... DO NOT QUIT"
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		log.Println("proceeding to run containers in isolated environment...")

		runCmd := exec.Command("docker-compose", "up", "-d")
		runCmd.Dir = rootPath
		output, err := runCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Output:", string(output))
			return err
		}

		log.Println("container running...", appPort, backendPort)
		stm = "Containers running..."
		errW = webhooks.PrecheckAction("finished", stm)
		if errW != nil {
			log.Println("api-service webhook failed", errW)
		}

		path = filepath.Join(rootPath, "server", "node_modules")
		utils.DeleteFile(path)

		path = filepath.Join(rootPath, "app", "node_modules")
		utils.DeleteFile(path)

		fmt.Printf("App is running on http://localhost:%d\n", appPort)

		return nil
	}
	cmd.Dir = path // same as cwd
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(appPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // optional, for prompts

	if err := cmd.Start(); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)
	return nil
}
