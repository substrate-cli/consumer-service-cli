package consumers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sshfz/consumer-service-substrate/internal/db"
	"github.com/sshfz/consumer-service-substrate/internal/helpers"
	"github.com/sshfz/consumer-service-substrate/internal/producers"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
)

type SpinRequest struct {
	UserId        string
	Message       string
	Prompt        string
	BackendPrompt string
}

var backendPort int
var appPort int

func SpinRequestConsumerFullStack(spinRequest SpinRequest) error {
	tempProjectName := "sleepyhead_slyme"
	homeDir, err := os.UserHomeDir()
	rootProjectPath := filepath.Join(homeDir, "Desktop", "substrate-home", tempProjectName)
	// os.MkdirAll(filepath.Join(rootProjectPath, "app"), os.ModePerm)

	exists, err := utils.DirExists(rootProjectPath)
	var serverCode map[string]any
	var appCode map[string]any
	if err != nil {
		log.Println("Error checking directory:", err)
	} else if exists {
		log.Println("Directory exists, skipping project and code generation.")
	} else {
		os.MkdirAll(filepath.Join(rootProjectPath, "server"), os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Directory does not exist")
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			path := filepath.Join(rootProjectPath)
			err = createNextApp(path) // creating a UI project
			if err != nil {
				log.Print("Unable to create next project")
			}
			log.Print("Next JS project initialised.")
		}()
		go func() {
			defer wg.Done()
			path := filepath.Join(rootProjectPath)
			err = createNodeJSProject(path) //create a nodejsproject
			if err != nil {
				log.Print("Unable to create node project")
			}
			log.Print("Server project initialised.")
		}()
		wg.Wait()
		log.Println("Setting up application cluster...")
		log.Print("Initiating code generation...")

		bPort, err := helpers.GetAvailablePort(3000, 3100)
		if err != nil {
			log.Fatalln("no free port available")
			return err
		}
		log.Println("backendPort =>", bPort)
		backendPort = bPort

		aPort, err := helpers.GetAvailablePort(backendPort+1, 3100)
		if err != nil {
			log.Fatalln("no free port available")
			return err
		}
		log.Println("appPort =>", aPort)
		appPort = aPort

		///generating backend structure ------
		portString := fmt.Sprintf("use port %d for this server", backendPort)
		serverPrompt := fmt.Sprintf("%s, %s", spinRequest.BackendPrompt, portString)
		backendStructure, err := producers.CallLLMNode(serverPrompt, *utils.GetServerStructCall())

		if err != nil {
			log.Println("there was a problem in generating backend struct.")
			log.Println(err)
			return err
		}

		backendStructure["appDescription"] = spinRequest.BackendPrompt
		errChan := make(chan error, 2)

		go func() {
			log.Println("Proceeding to generate server code....")
			jsonBytes, err := json.Marshal(backendStructure)
			if err != nil {
				log.Println("Error parsing backend structure")
				log.Println(err)
				errChan <- err
			}
			serverCode, err = producers.CallLLMNode(string(jsonBytes), *utils.GetServerGenCall())
			if err != nil {
				log.Println("Error generating node js code")
				log.Println(err)
				errChan <- err
			}
			errChan <- nil
		}()

		go func() {
			log.Println("proceeding to generate app code...")
			serverStruct := backendStructure["server"].(map[string]any)
			apis := serverStruct["apis"].(map[string]any)
			baseApiUrl := fmt.Sprintf("http://localhost:%d", backendPort)
			apis["userPrompt"] = spinRequest.Prompt
			apis["baseApiUrl"] = baseApiUrl
			jsonBytes, err := json.Marshal(apis)
			if err != nil {
				log.Println("error parsing backend struct")
				log.Println(err)
				errChan <- err
			}
			appCode, err = producers.CallLLMNode(string(jsonBytes), *utils.GetAppGenFSCall())
			if err != nil {
				log.Println("Error generating app code")
				log.Println(err)
				errChan <- err
			}
			errChan <- nil
		}()

		var hasError bool
		for i := 0; i < 2; i++ {
			if err := <-errChan; err != nil {
				log.Printf("❌ Error: %v\n", err)
				hasError = true
			}
		}

		if hasError {
			log.Fatal("💥 One or more tasks failed")
		} else {
			log.Println("✅ Both projects created successfully")
		}

		log.Println("Server code successfully generated")
		log.Println("App Code successfully generated")
	}

	if err != nil {
		log.Fatal("Failed to spin project:", err)
		return err
	}

	log.Println("*********Code generation complete*********")
	log.Println("Proceeding to write codes....")

	errChan := make(chan error, 2)

	go func() {
		result := appCode
		path := filepath.Join(rootProjectPath, "app")
		errChan <- generateCode(path, result["app"].(map[string]any))
	}()

	go func() {
		result := serverCode
		path := filepath.Join(rootProjectPath, "server")
		errChan <- generateCode(path, result["server"].(map[string]any))
	}()

	var hasError bool
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			log.Printf("❌ Error: %v\n", err)
			hasError = true
		}
	}
	if hasError {
		log.Fatal("One or more tasks failed")
	} else {
		log.Println("✅ Cluster running successfully.")
	}

	/// building and running the project on different ports.-----

	log.Print("Initiating build and starting projects...")
	err = runProject(rootProjectPath, true)
	if err != nil {
		return err
	}

	// ---- calling api-service api to open websocket ------
	// sendPorts := map[string]interface{}{
	// 	"appPort":     appPort,
	// 	"backendPort": backendPort,
	// }
	// err = helpers.CodeGenerationAction("finished", sendPorts)
	// if err != nil {
	// 	log.Println("Error calling code generation webhook")
	// 	return err
	// }
	return nil
}

func createNextApp(projectPath string) error {
	// basePath := filepath.Join(projectPath)

	// 🔹 Create Next.js app using npx (must be installed globally)
	cmd := exec.Command(
		"npx", "create-next-app@latest",
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

func generateCode(appPath string, data map[string]interface{}) error {
	rawMap, ok := data["fileStructure"].(map[string]interface{})
	rawLibraries, ok2 := data["libraries"].([]interface{})
	log.Print("kkkkkkkk", rawLibraries)
	var libraries []string
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
		log.Fatal("fileStructure is not a map[string]interface{}")
	}

	fileMap := make(map[string]string)

	for path, content := range rawMap {

		if m, ok := content.(map[string]interface{}); ok {
			if content, exists := m["code"]; exists {
				strContent, ok := content.(string)
				if !ok {
					log.Fatalf("content for path %s is not a string", path)
				}
				fileMap[path] = strContent
			}
		} else {
			fmt.Printf("Path %s is not a map, got %T\n", path, content)
		}

	}

	err := installLibraries(appPath, libraries)
	if err != nil {
		log.Println("Failed to install libraries in node js server")
		return err
	}

	log.Println("✅ fileMap ready:", fileMap)
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

			jsonData, err := json.Marshal(fileContentMapRedis)
			if err != nil {
				log.Fatal("Error marshaling map:", err)
			}

			//saving to redis ----
			db.SaveRedis("sleepyhead_slyme:structure", string(jsonData))
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

func runProject(rootPath string, isbackend bool) error {
	type error interface {
		Error() string
	}
	var err error
	if isbackend {
		err = runServer(rootPath)
		if err != nil {
			log.Fatal("Failed to start server")
			return err
		}
	}
	err = runApp(rootPath)
	if err != nil {
		log.Fatal("Failed to start app")
		return err
	}
	return nil
}

func runServer(rootPath string) error {
	path := filepath.Join(rootPath, "server")
	cmd := exec.Command("npm", "run", "dev")
	// port, err := helpers.GetAvailablePort(3000, 3100)
	// if err != nil {
	// 	log.Println("Terminating execution, no free port available.")
	// 	return err
	// }
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

func runApp(rootPath string) error {
	path := filepath.Join(rootPath, "app")
	cmd := exec.Command("npm", "run", "dev")
	log.Println("starting next js server...")
	// if err != nil {
	// 	log.Println("Terminating execution, no free port available.")
	// 	return err
	// }
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

func RunBuildCommand(rootPath string) ([]map[string]string, error) {
	log.Println("executing build just to check everything's smooth...")
	var out bytes.Buffer
	var stderr bytes.Buffer
	////////webhook to ve called later ------
	path := filepath.Join(rootPath, "app")
	cmd := exec.Command("npm", "run", "build")

	cmd.Dir = path
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	errorsMatch := []map[string]string{}

	if err := cmd.Run(); err != nil {
		output := stderr.String()
		re := regexp.MustCompile(`(\.?\/[^\s:()]+)[(:](\d+):(\d+)`)
		matches1 := re.FindAllStringSubmatch(output, -1)

		// Get the actual error message (rest of output after file line)
		if len(matches1) > 0 {
			log.Println(len(matches1), "file")

			blocks := strings.Split(output, "Caused by:")
			for _, block := range blocks {
				matches := re.FindAllStringSubmatch(block, -1)
				if len(matches) > 0 {
					errorMap := make(map[string]string)
					filePath := matches[0][1]
					line := matches[0][2]
					col := matches[0][3]

					fmt.Println("File:", filePath, "Line:", line, "Col:", col)
					// fmt.Println("Error snippet:\n", block)
					fmt.Println("---------------")
					var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)
					cleanPath := strings.TrimSpace(ansi.ReplaceAllString(filePath, ""))
					log.Println(cleanPath, "ffffffffff")
					relativePath := getRelativePathSimple(cleanPath, path)
					code, err := os.ReadFile(cleanPath)
					if err != nil {
						log.Println("Unable to read file")
						log.Println(err)
						return nil, err
					}

					errorMap["filePath"] = string(filePath)
					errorMap["error_line_number"] = string(line)
					errorMap["error"] = string(block)
					errorMap["relativeFilePath"] = relativePath
					errorMap["actualCode"] = string(code)

					//extracting imports from the file ----

					///
					errorsMatch = append(errorsMatch, errorMap)
					// appending also the files which are related
				}
			}
			return errorsMatch, nil
		} else {
			return []map[string]string{}, nil
		}
	}

	return errorsMatch, nil
}

func getRelativePathSimple(absolutePath, projectRoot string) string {
	// Clean paths
	cleanAbsolute := filepath.Clean(absolutePath)
	cleanRoot := filepath.Clean(projectRoot)

	// Replace the project root part with empty string
	if strings.HasPrefix(cleanAbsolute, cleanRoot) {
		relative := strings.TrimPrefix(cleanAbsolute, cleanRoot)
		relative = strings.TrimPrefix(relative, "/") // Remove leading slash
		return relative
	}

	// If not under project root, try parent directory
	parentRoot := filepath.Dir(cleanRoot)
	if strings.HasPrefix(cleanAbsolute, parentRoot) {
		relative := strings.TrimPrefix(cleanAbsolute, parentRoot)
		relative = strings.TrimPrefix(relative, "/")
		return relative
	}

	return filepath.Base(cleanAbsolute)
}

func RunBuildCommand2(rootPath string) ([]map[string]string, error) {
	var out, stderr bytes.Buffer
	path := filepath.Join(rootPath, "app")

	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = path
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	errorMap := make(map[string]string)
	errorsMatch := []map[string]string{}

	if err := cmd.Run(); err != nil {
		log.Println("Build failed:", stderr.String())
		output := stderr.String() + out.String()

		// Regex to capture relative path, line/column, and absolute path + message
		re := regexp.MustCompile(`(?m)\./(.*\.(css|ts|tsx|js|jsx)):(\d+):(\d+)\s*\nSyntax error:\s*(/.*?)(\sThe .+)`)
		matches := re.FindAllStringSubmatch(output, -1)

		if len(matches) == 0 {
			log.Println("no build issues found.")
			return []map[string]string{}, nil
		}

		for _, m := range matches {
			relativePath := "/" + m[1] // keep relative path
			line := m[3]
			column := m[4]
			absolutePath := m[5]                // absolute file path
			errorMsg := strings.TrimSpace(m[6]) // actual error message

			fmt.Printf("File: %s\nLine: %s, Column: %s\nAbsolutePath: %s\nErrorMessage: %s\n\n",
				relativePath, line, column, absolutePath, errorMsg)

			code, err := os.ReadFile(absolutePath)
			if err != nil {
				log.Println(err)
				return []map[string]string{}, err
			}
			log.Println(string(code))

			errorMap["filePath"] = relativePath
			errorMap["error"] = errorMsg
			errorsMatch = append(errorsMatch, errorMap)

			// helpers.CallAnthropicError(errorsMatch)
		}
	}
	return errorsMatch, nil
}
func RunBuildCommand3(rootPath string) ([]map[string]string, error) {
	var out, stderr bytes.Buffer
	path := filepath.Join(rootPath, "app")

	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = path
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	errorMap := make(map[string]string)
	errorsMatch := []map[string]string{}

	if err := cmd.Run(); err != nil {
		log.Println("Build failed:", stderr.String())
		output := stderr.String() + out.String()

		// Regex to capture relative path, line/column, and absolute path + message
		re := regexp.MustCompile(`(?m)^(\.\/[^\n]+\.(?:css|ts|tsx|js|jsx))\s*\nError:\s+x\s+([^\n]+)\n\s+,-\[([^\]]+):(\d+):(\d+)\]`)
		matches := re.FindAllStringSubmatch(output, -1)

		log.Println(matches, "mmmmmmmmmm")
		if len(matches) == 0 {
			// log.Println("no build issues found.")
			altRe := regexp.MustCompile(`(?m)(\.\/[^\s]+\.(?:css|ts|tsx|js|jsx)).*?Error.*?(\d+):(\d+).*?\n.*?([^\n]+)`)
			matches = altRe.FindAllStringSubmatch(output, -1)
			log.Println("Alternative matches:", matches)
		}

		if len(matches) == 0 {
			log.Println("no build issues found.")
			return []map[string]string{}, nil
		}

		for _, m := range matches {
			relativePath := "/" + m[1] // keep relative path
			line := m[3]
			column := m[4]
			absolutePath := m[5]                // absolute file path
			errorMsg := strings.TrimSpace(m[6]) // actual error message

			fmt.Printf("File: %s\nLine: %s, Column: %s\nAbsolutePath: %s\nErrorMessage: %s\n\n",
				relativePath, line, column, absolutePath, errorMsg)

			code, err := os.ReadFile(absolutePath)
			if err != nil {
				log.Println(err)
				return []map[string]string{}, err
			}
			log.Println(string(code))

			errorMap["filePath"] = relativePath
			errorMap["error"] = errorMsg
			errorsMatch = append(errorsMatch, errorMap)

			// helpers.CallAnthropicError(errorsMatch)
		}
	}
	return errorsMatch, nil
}
