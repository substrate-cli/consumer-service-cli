package consumers

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/sshfz/consumer-service-substrate/internal/db"
	"github.com/sshfz/consumer-service-substrate/internal/helpers"
	"github.com/sshfz/consumer-service-substrate/internal/producers"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
	"github.com/sshfz/consumer-service-substrate/internal/webhooks"
)

func SpinRequestConsumer(spinRequest SpinRequest) error {
	tempProjectName := "sleepyhead_slyme"
	homeDir, err := os.UserHomeDir()
	rootProjectPath := filepath.Join(homeDir, "Desktop", "substrate-home", tempProjectName)

	exists, err := utils.DirExists(rootProjectPath)
	if err != nil {
		log.Println("Error checking directory:", err)
	} else if exists {
		log.Println("Directory exists, skipping project and code generation.")
	} else {
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Directory does not exist")
		os.MkdirAll(filepath.Join(rootProjectPath, "app"), os.ModePerm)
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

		log.Print("Initiating code generation...")

		aPort, err := helpers.GetAvailablePort(3000, 3100)
		if err != nil {
			log.Fatalln("no free port available")
			return err
		}
		log.Println("appPort =>", aPort)
		appPort = aPort

		result, err := producers.CallLLMNode(spinRequest.Prompt, *utils.GetAppGenCall())
		if err != nil {
			log.Println("Error during app generation")
			return err
		}

		err = webhooks.PrecheckAction("finished", "code generation for next app completed.")
		if err != nil {
			log.Println("api-service webhook failed")
			return err
		}

		errChan := make(chan error, 1)

		go func() {
			path := filepath.Join(rootProjectPath, "app")
			errChan <- generateCode(path, result["app"].(map[string]any))
		}()

		var hasError bool
		for i := 0; i < 1; i++ {
			if err := <-errChan; err != nil {
				log.Printf("Error: %v\n", err)
				hasError = true
			}
		}

		if hasError {
			log.Fatal("One or more tasks failed")
		} else {
			log.Println("Next js project created successfully")

			err = webhooks.PrecheckAction("finished", "code written succesfully.")
			if err != nil {
				log.Println("api-service webhook failed")
				return err
			}
		}

	}

	if err != nil {
		log.Println(err)
		return err
	}
	/// building and running the project on different ports.-----

	log.Print("Initiating build and starting projects...")
	err = runProject(rootProjectPath, false)
	if err != nil {
		return err
	}

	db.SaveRedis(tempProjectName, "running")
	// ---- calling api-service api to open websocket ------
	sendPorts := map[string]interface{}{
		"appPort": appPort,
	}
	err = webhooks.CodeGenerationAction("finished", sendPorts)
	if err != nil {
		log.Println("Error calling code generation webhook")
		return err
	}

	log.Println("code generation webhook proccessed")

	//running build -----
	// RunBuildCommand2(rootProjectPath)
	log.Println("Running build -------------------------------------")
	// errorsMatch, err := RunBuildCommand(rootProjectPath)
	// if err != nil {
	// 	log.Println("Error found in build")
	// 	return err
	// }

	// if len(errorsMatch) > 0 {
	// 	data, err := helpers.CallAnthropicError(errorsMatch)

	// 	if err != nil {
	// 		log.Println("Error in build llm call")
	// 		log.Println(err)
	// 	}
	// 	path := filepath.Join(rootProjectPath, "app")
	// 	err = FixBuildCode(path, data["app"].(map[string]interface{}))
	// 	if err != nil {
	// 		log.Println("Error fixing build issues.")
	// 		log.Println(err)
	// 	}
	// }
	////build code ends ----------

	return nil
}

func FixBuildCode(appPath string, data map[string]interface{}) error {
	rawMap, ok := data["fileStructure"].(map[string]interface{})
	if !ok {
		log.Fatal("fileStructure is not a map[string]interface{}")
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
