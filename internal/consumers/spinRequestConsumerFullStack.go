package consumers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/substrate-cli/consumer-service-cli/internal/db"
	"github.com/substrate-cli/consumer-service-cli/internal/helpers"
	"github.com/substrate-cli/consumer-service-cli/internal/producers"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
	"github.com/substrate-cli/consumer-service-cli/internal/webhooks"
)

type SpinRequest struct {
	UserId        string
	Message       string
	Prompt        string
	BackendPrompt string
	ApiKey        string
	ClusterName   string
	Model         string
	IsClone       bool
}

var backendPort int
var appPort int

func SpinRequestConsumerFullStack(spinRequest SpinRequest) error {
	clusterName := spinRequest.ClusterName
	rootProjectPath, err := utils.GetHomeDirectory()
	if err != nil {
		return err
	}
	rootProjectPath = filepath.Join(rootProjectPath, "substrate-home", clusterName)
	// os.MkdirAll(filepath.Join(rootProjectPath, "app"), os.ModePerm)

	exists, err := utils.DirExists(rootProjectPath)
	var serverCode map[string]any
	var appCode map[string]any
	if err != nil {
		log.Println("Error checking directory:", err)
		errW := webhooks.ErrorAction("finished", "failed to create cluster", "error checking directory")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	} else if exists {
		log.Println("Directory exists, skipping project and code generation.")
		errW := webhooks.ErrorAction("finished", "Directory already exists pls choose a different name for the project.", "cluster init failed")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return errors.New("Directory already exists pls choose a different name for the project.")
	} else {
		os.MkdirAll(filepath.Join(rootProjectPath, "server"), os.ModePerm)
		if err != nil {
			log.Println(err)
			errW := webhooks.ErrorAction("finished", "failed to create cluster", err.Error())
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}

		log.Println("Directory does not exist")
		//

		//
		log.Println("Setting up application cluster...")
		log.Print("Initiating code generation...")

		bPort, err := helpers.GetAvailablePort(3000, 3100)
		if err != nil {
			log.Println("no free port available")
			errW := webhooks.ErrorAction("finished", "No free port available", "No free port available")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
		log.Println("backendPort =>", bPort)
		backendPort = bPort

		aPort, err := helpers.GetAvailablePort(backendPort+1, 3100)
		if err != nil {
			errW := webhooks.ErrorAction("finished", "error creating cluster", "error getting port")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			log.Println("no free port available")
			return err
		}
		log.Println("appPort =>", aPort)
		appPort = aPort

		///generating backend structure ------
		portString := fmt.Sprintf("use port %d for this server", backendPort)
		serverPrompt := fmt.Sprintf("%s, %s", spinRequest.BackendPrompt, portString)
		str := make(map[string]string)
		key := utils.GetCLIApiKey()
		if key != nil {
			str["apiKey"] = *key
		}
		str["prompt"] = serverPrompt
		str["model"] = *utils.GetModel()
		jsonBytes, err := json.Marshal(str)
		if err != nil {
			log.Println("error parsing prompt struct")
			log.Println(err)
		}
		errW := webhooks.PrecheckAction("finished", "proceeding to generate server struct... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		backendStructure, err := producers.CallLLMNode(string(jsonBytes), *utils.GetServerStructCall())

		if err != nil {
			log.Println("there was a problem in generating backend struct.")
			log.Println(err)
			errW := webhooks.ErrorAction("finished", "error creating cluster", "there was a problem in generating backend struct")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
		errW = webhooks.PrecheckAction("finished", "backend struct generated succesfully.")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		backendStructure["appDescription"] = spinRequest.BackendPrompt
		backendStructure["model"] = *utils.GetModel()
		if key != nil {
			backendStructure["apiKey"] = *key
		}
		errChan := make(chan error, 2)

		errW = webhooks.PrecheckAction("finished", "Proceeding to assign LLM... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		go func() {
			log.Println("Proceeding to generate server code... DO NOT QUIT")
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
				errW := webhooks.ErrorAction("finished", "error creating cluster", err.Error())
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				errChan <- err
			}
			errW = webhooks.PrecheckAction("finished", "server code generated succesfully. DO NOT QUIT")
			if errW != nil {
				log.Println("api-service webhook failed")
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
			apis["model"] = *utils.GetModel()
			if key != nil {
				apis["apiKey"] = *key
			}
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
				errW := webhooks.ErrorAction("finished", "error creating cluster", err.Error())
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				errChan <- err
			}
			errW = webhooks.PrecheckAction("finished", "app code generated succesfully. DO NOT QUIT")
			if errW != nil {
				log.Println("api-service webhook failed")
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
			log.Println("One or more tasks failed")
			errW := webhooks.ErrorAction("finished", "error creating cluster", "One or more tasks failed")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return errors.New("One or more tasks failed on LLM call")
		} else {
			log.Println("✅ Both projects created successfully")
		}

		errW = webhooks.PrecheckAction("finished", "Code Generation complete, Initialising cluster... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		log.Println("Server code successfully generated")
		log.Println("App Code successfully generated")
	}

	if err != nil {
		log.Println("Failed to spin project:", err)
		errW := webhooks.ErrorAction("finished", "error creating cluster", "failed to spin cluster")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}

	//
	var wg sync.WaitGroup
	wg.Add(2)
	errW := webhooks.PrecheckAction("finished", "Creating Cluster... DO NOT QUIT")
	if errW != nil {
		log.Println("api-service webhook failed")
	}
	go func() {
		defer wg.Done()
		path := filepath.Join(rootProjectPath)
		err = createNextApp(path) // creating a UI project
		if err != nil {
			log.Print("Unable to create next project")
		}
		err = webhooks.PrecheckAction("finished", "next app initialised succesfully.")
		if err != nil {
			log.Println("api-service webhook failed")
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
		errW := webhooks.PrecheckAction("finished", "node js server initialised succesfully.")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		log.Print("Server project initialised.")
	}()
	wg.Wait()
	//

	errW = webhooks.PrecheckAction("finished", "Cluster initialised... warming up...")
	if errW != nil {
		log.Println("api-service webhook failed")
	}

	log.Println("*********Code generation complete*********")
	log.Println("Proceeding to write codes....")

	errChan := make(chan error, 2)

	go func() {
		result := appCode
		path := filepath.Join(rootProjectPath, "app")
		initCommands := [][]string{
			{"npx", "shadcn@latest", "init", "-d"},
		}
		errChan <- generateCode(path, result["app"].(map[string]any), *utils.GetDockerNext(), spinRequest.ClusterName, initCommands)
	}()

	go func() {
		result := serverCode
		path := filepath.Join(rootProjectPath, "server")
		initCommands := [][]string{
			{"npx", "prisma", "init"},
		}
		errChan <- generateCode(path, result["server"].(map[string]any), *utils.GetDockerNode(), spinRequest.ClusterName, initCommands)
	}()

	var hasError bool
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			log.Printf("❌ Error: %v\n", err)
			hasError = true
		}
	}
	if hasError {
		log.Println("One or more tasks failed")
		errW := webhooks.ErrorAction("finished", "error creating cluster", "one or more tasks failed")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	} else {
		log.Println("✅ Cluster running successfully.")
		errW := webhooks.PrecheckAction("finished", "Cluster initialised succesfully.")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
	}

	/// building and running the project on different ports.-----

	log.Print("Initiating build and starting projects...")
	err = runProject(rootProjectPath, true, spinRequest.ClusterName, true)
	if err != nil {
		return err
	}

	db.SaveRedis(clusterName, "running")

	sendPorts := map[string]interface{}{
		"appPort":    appPort,
		"serverPort": backendPort,
	}
	errW = webhooks.CodeGenerationAction("finished", sendPorts)
	if errW != nil {
		log.Println("Error calling code generation webhook")
	}

	cluster := map[string]interface{}{
		"clusterName": spinRequest.ClusterName,
	}
	errW = webhooks.CodeGenerationAction("finished", cluster)
	if errW != nil {
		log.Println("Error calling code generation webhook")
	}

	log.Println("cluster running...")

	return nil
}
