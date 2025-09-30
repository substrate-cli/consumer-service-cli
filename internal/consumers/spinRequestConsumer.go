package consumers

import (
	"encoding/json"
	"errors"
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

func SpinRequestConsumerApp(spinRequest SpinRequest) error {
	clusterName := spinRequest.ClusterName
	rootProjectPath, err := utils.GetHomeDirectory()
	if err != nil {
		return err
	}
	rootProjectPath = filepath.Join(rootProjectPath, "substrate-home", clusterName)

	exists, err := utils.DirExists(rootProjectPath)
	if err != nil {
		log.Println("Error checking directory:", err)
	} else if exists {
		log.Println("Directory exists, skipping project and code generation.")
		return errors.New("Directory already exists pls choose a different for the project.")
	} else {
		if err != nil {
			errW := webhooks.ErrorAction("finished", "failed to create cluster", "error checking directory")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			log.Println(err)
			return err
		}

		log.Println("Directory does not exist")
		os.MkdirAll(filepath.Join(rootProjectPath, "app"), os.ModePerm)

		//

		////

		log.Print("Initiating code generation...")
		errW := webhooks.PrecheckAction("finished", "initiating code generation... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		aPort, err := helpers.GetAvailablePort(3000, 3100)
		if err != nil {
			log.Println("no free port available")
			errW := webhooks.ErrorAction("finished", "cluster creation failed", "no free port available")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
		log.Println("appPort =>", aPort)
		appPort = aPort
		str := make(map[string]string)
		key := utils.GetCLIApiKey()
		if key != nil {
			str["apiKey"] = *key
		}
		str["prompt"] = spinRequest.Prompt
		str["model"] = *utils.GetModel()
		jsonBytes, err := json.Marshal(str)
		if err != nil {
			log.Println("error parsing prompt struct")
			log.Println(err)
			errW := webhooks.ErrorAction("finished", "cluster creation failed", "error parsing prompt struct")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}

		errW = webhooks.PrecheckAction("finished", "Assigning LLM... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		routingKey := *utils.GetAppGenCall()
		if spinRequest.IsClone {
			routingKey = *utils.GetCloneAppGen()
		}
		result, err := producers.CallLLMNode(string(jsonBytes), routingKey)
		if err != nil {
			log.Println("Error during app generation")
			errW := webhooks.ErrorAction("finished", "cluster creation failed", "error during app generation")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}

		errW = webhooks.PrecheckAction("finished", "code generation for next app completed.")
		if errW != nil {
			log.Println("api-service webhook failed")
			return err
		}

		///
		errW = webhooks.PrecheckAction("finished", "Starting Cluster... DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			path := filepath.Join(rootProjectPath)
			err = createNextApp(path) // creating a UI project
			if err != nil {
				log.Print("Unable to create next project")
				log.Println("Error =>", err)
			}
			log.Print("Next JS project initialised.")
		}()

		wg.Wait()
		if err != nil {
			errW = webhooks.ErrorAction("finished", "cluster creation failed", "one or more tasks failed")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
		errW = webhooks.PrecheckAction("finished", "Next App Initialised, Proceeding for code generation, DO NOT QUIT")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		///

		errChan := make(chan error, 1)

		go func() {
			path := filepath.Join(rootProjectPath, "app")
			errChan <- generateCode(path, result["app"].(map[string]any), *utils.GetDockerNext(), spinRequest.ClusterName)
		}()

		var hasError bool
		for i := 0; i < 1; i++ {
			if err := <-errChan; err != nil {
				log.Printf("Error: %v\n", err)
				hasError = true
			}
		}

		if hasError {
			log.Println("One or more tasks failed")
			errW = webhooks.ErrorAction("finished", "cluster creation failed", "one or more tasks failed")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return errors.New("One or more tasks failed on LLM call")
		} else {
			log.Println("Next js project created successfully")
			errW = webhooks.PrecheckAction("finished", "code written succesfully.")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
		}
	}

	if err != nil {
		log.Println(err)
		return err
	}
	/// building and running the project on different ports.-----

	log.Print("Initiating build and starting projects...")
	err = runProject(rootProjectPath, false, spinRequest.ClusterName, false)
	if err != nil {
		errW := webhooks.ErrorAction("finished", "cluster created, but failed to run.", "failed to run project")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}

	db.SaveRedis(clusterName, "running")
	// ---- calling api-service api to open websocket ------
	sendPorts := map[string]interface{}{
		"appPort": appPort,
	}
	errW := webhooks.CodeGenerationAction("finished", sendPorts)
	if errW != nil {
		log.Println("Error calling code generation webhook")
		return err
	}

	log.Println("code generation webhook processed")

	return nil
}

func FixBuildCode(appPath string, data map[string]interface{}) error {
	rawMap, ok := data["fileStructure"].(map[string]interface{})
	if !ok {
		log.Println("fileStructure is not a map[string]interface{}")
		errW := webhooks.ErrorAction("finished", "error creating cluster", "invalid file structure")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return errors.New("invalid file structure")
	}

	for filePath, content := range rawMap {
		relPath := filepath.Join(appPath, filePath)
		err := utils.WriteFiles(relPath, content.(string))
		if err != nil {
			return err
		}
	}

	return nil
}
