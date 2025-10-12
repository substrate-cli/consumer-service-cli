package consumers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/substrate-cli/consumer-service-cli/internal/headless"
	"github.com/substrate-cli/consumer-service-cli/internal/helpers"
	"github.com/substrate-cli/consumer-service-cli/internal/interfaces"
	"github.com/substrate-cli/consumer-service-cli/internal/llm"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
	"github.com/substrate-cli/consumer-service-cli/internal/webhooks"
)

// handleTask processes the received task message
func HandleSpinConsumer(body []byte) error {

	log.Printf("inside handle-spin-consumer")

	var payload SpinRequest

	err := json.Unmarshal(body, &payload)
	if err != nil {
		log.Println("failed to decode json")
		errW := webhooks.ErrorAction("failed", "failed to decode json", err.Error(), false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}
	log.Println("User prompt => ", payload.Prompt)
	if payload.ApiKey != "" || len(payload.ApiKey) != 0 {
		utils.SetCLIApiKey(payload.ApiKey)
	}

	payload.ClusterName = strings.TrimSpace(payload.ClusterName)
	if payload.ClusterName == "" || len(payload.ClusterName) == 0 {
		payload.ClusterName = helpers.GenerateProjectName()
	}

	payload.Model = strings.ToLower(payload.Model)
	payload.Model = strings.TrimSpace(payload.Model)
	err = helpers.SpecifyModel(payload.Model)
	if err != nil {
		errW := webhooks.ErrorAction("failed", "invalid model reference", err.Error(), false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}

	model := utils.GetModel()
	client, err := llm.NewLLMClient(*model)
	if err != nil {
		log.Println("Error setting provider")
		log.Println(err)
		errW := webhooks.ErrorAction("failed", "Error setting provider", err.Error(), false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}

	//precheck -------
	type Response struct {
		Is_valid_prompt  bool   `json:"is_valid_prompt"`
		Response         string `json:"response"`
		Reason           string `json:"reason"`
		Requires_backend bool   `json:"requires_backend"`
		Type             string `json:"type"`
		Clone_type       string `json:"clone_type"`
		Url              string `json:"url"`
		RepoUrl          string `json:"repoUrl"`
		Author           string `json:"author"`
		Repo_name        string `json:"repo_name"`
	}
	var response Response
	res, err := client.CallPrecheck(payload.Prompt)
	if err != nil {
		log.Println("Error in llm precheck.")
		log.Println(err)
		errW := webhooks.ErrorAction("failed", "error creating cluster", err.Error(), false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}
	err = json.Unmarshal([]byte(res), &response)
	if err != nil {
		log.Println("Error in decoding llm precheck response", err)
		errW := webhooks.ErrorAction("failed", "error creating cluster", err.Error(), false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		return err
	}
	if !response.Is_valid_prompt {
		///call webhook in api-server for failed attempt
		errW := webhooks.ErrorAction("failed", response.Reason, "invalid prompt", false)
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		log.Println("invalid prompt detected for app generation, proceeding to call webhook in api-server")
		return errors.New("invalid prompt detected for app generation")
	}
	///calling webhook for successful prechcek -----
	log.Println("Anthropic Precheck passed, proceeding for code generation...")

	errW := webhooks.PrecheckAction("finished", response.Response)
	if errW != nil {
		log.Println("api-service webhook failed")
	}
	log.Println("Proceeding for code generation...")
	log.Println("Initiating code generation for => ", payload.Prompt)

	//
	if response.Type == "prompt" {
		if !response.Requires_backend {
			log.Println("Backend not required for cluster")
			log.Println("Proceeding to generate next js code generation.")
			err = SpinRequestConsumerApp(payload)
			if err != nil {
				const msg = "Cluster creation failed"
				errW = webhooks.ErrorAction("finished", msg, "failed to spin cluster", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				return err
			}
			return nil
		} else {
			log.Println("Backend required for cluster")
			log.Println("Proceeding to generate full stack application")
			///generating backend prompt -------
			backendStructPrompt, err := client.CallConstructBackendPrompt(payload.Prompt)
			if err != nil {
				log.Println("Error constrcuting backend prompt")
				errW = webhooks.ErrorAction("finished", "Error in server generation", "failed to constrcut backend prompt", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				return err
			}
			errW = webhooks.PrecheckAction("finished", "backend prompt generated.")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			if err != nil {
				return err
			}
			log.Println("Backend Struct Prompt => ", backendStructPrompt)
			payload.BackendPrompt = backendStructPrompt
			err = SpinRequestConsumerFullStack(payload)
			if err != nil {
				errW = webhooks.ErrorAction("finished", err.Error(), "Failed to generate clone", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}

				return err
			}
			return nil
		}
		//
	}
	if response.Type == "clone" {
		if response.Clone_type == "repo" { //for github scan -----
			log.Println("Github repository action detected")
			errW = webhooks.PrecheckAction("finished", "Navigating "+response.RepoUrl)
			if errW != nil {
				log.Println("api-service webhook failed")
			}

			repoUrl := response.RepoUrl
			author := response.Author
			repoName := response.Repo_name

			visionResponse, err := headless.GenerateClonePromptByVision(repoUrl, client, true)
			if err != nil {
				errW = webhooks.ErrorAction("finished", err.Error(), "Failed to generate clone", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				return err
			}

			errW = webhooks.PrecheckAction("finished", "Checking for more details in repository... DO NOT QUIT")
			if errW != nil {
				log.Println("api-service webhook failed")
			}

			if !visionResponse["clonable"].(bool) { //11111111
				reason := visionResponse["reasoning"]
				errW = webhooks.ErrorAction("finished", reason.(string), reason.(string), false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}

				return errors.New("Unable to clone github repo")
			}

			if !visionResponse["is_repository"].(bool) { /// only 1% chance ----
				errW = webhooks.ErrorAction("finished", "Invalid github repo", "Invalid github repo", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				return errors.New("Unable to clone github repo")
			}

			defBranch := visionResponse["default_branch"] ///
			if defBranch == nil {
				errW = webhooks.ErrorAction("finished", "Unable to fetch default branch from repository scan, trying to fetch github tree for more details...", "default branch is null", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				return errors.New("cluster init failed")
			}
			descriptionFromVision := visionResponse["description"]
			liveUrl := visionResponse["live_url"]

			if descriptionFromVision == nil {
				errW = webhooks.ErrorAction("finished", "Unable to fetch description from repository scan, trying to fetch github tree for more details...", "Invalid github repo", true)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
			}

			errW = webhooks.PrecheckAction("finished", descriptionFromVision.(string))
			if errW != nil {
				log.Println("api-service webhook failed")
			}

			tree, err := headless.FetchRepoTree(repoName, author, defBranch.(string))
			if err != nil {
				errW = webhooks.ErrorAction("finished", err.Error(), "Failed to init cluster", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				log.Println(err)

				return err
			}

			errW = webhooks.PrecheckAction("finished", "Files fetched from repository...")
			if errW != nil {
				log.Println("api-service webhook failed")
			}

			if len(tree.Tree) == 0 {
				errW = webhooks.ErrorAction("finished", "no files found in repository", "Failed to init cluster", false)
				if errW != nil {
					log.Println("api-service webhook failed")
				}
				log.Println(err)
				return errors.New("no files found in repository")
			}

			arr := make([]string, 0)
			limit := len(tree.Tree)
			if limit > 100 {
				limit = 100
			}
			for _, val := range tree.Tree[:limit] {
				s := strings.ReplaceAll(val.Path, " ", "")
				if val.Path != "" && len(s) > 0 {
					arr = append(arr, val.Path)
				}
			}
			jsonBytes, err := json.Marshal(arr)
			finalResp := ""
			if len(arr) > 0 {
				finalResp, err = client.CallGithubTreeScan(string(jsonBytes))

				if err != nil {
					errW = webhooks.ErrorAction("finished", err.Error(), "Failed to init cluster", false)
					if errW != nil {
						log.Println(errW.Error())
						log.Println("api-service webhook failed")
					}
					return err
				}
			}

			errW = webhooks.PrecheckAction("finished", "Sanning completed... DO NOT QUIT")
			if errW != nil {
				log.Println("api-service webhook failed")
			}

			var treeScanResponse map[string]any
			err = json.Unmarshal([]byte(finalResp), &treeScanResponse)
			if err != nil {
				log.Println(err)
				errW = webhooks.ErrorAction("finished", err.Error(), "Failed to init cluster", false)
				if errW != nil {
					log.Println(errW.Error())
					log.Println("api-service webhook failed")
				}
				return err
			}

			if !treeScanResponse["is_clonable"].(bool) {
				log.Println("Github tree scan says not clonable!")
				errW = webhooks.ErrorAction("finished", treeScanResponse["reason"].(string), "Failed to init cluster", false)
				if errW != nil {
					log.Println(treeScanResponse["reason"].(string))
					log.Println("api-service webhook failed")
				}
				return errors.New("Repository not clonable, quitting")
			}

			descriptionFromTreeScan := treeScanResponse["description"]
			//  descriptionFromVision, liveurl, descriptionFromTreeScan
			if liveUrl != nil && liveUrl != "" {
				log.Println("live url found")
				log.Println("proceeding to intiate cluster based on url")

				scrapeAndCloneUrl(liveUrl.(string), payload, client)
				return nil
			}

			if descriptionFromVision != nil && descriptionFromVision != "" {
				//prompt gen
				log.Println("description from vision found")
				log.Println("proceeding to intiate cluster based description from vision")

				log.Println("Proceeding to generate next js code generation.")
				promptGen, err := client.CallPrePromptForGithubClone(descriptionFromVision.(string))
				if err != nil {
					log.Println(err.Error())
					errW = webhooks.ErrorAction("finished", err.Error(), "Failed to init cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return err
				}
				if promptGen == "" {
					log.Println("failed to generate prompt")
					errW = webhooks.ErrorAction("finished", "failed to generate prompt", "Failed to init cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return errors.New("failed to generate prompt")
				}
				payload.Prompt = promptGen
				err = SpinRequestConsumerApp(payload)
				if err != nil {
					const msg = "Cluster creation failed"
					errW = webhooks.ErrorAction("finished", msg, "failed to spin cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return err
				}
				return nil
			}

			if descriptionFromTreeScan != nil && descriptionFromTreeScan != "" {
				//prompt gen
				log.Println("description from tree scan found")
				log.Println("proceeding to intiate cluster based description from tree scan")

				log.Println("Proceeding to generate next js code generation.")

				promptGen, err := client.CallPrePromptForGithubClone(descriptionFromTreeScan.(string))
				if err != nil {
					log.Println(err.Error())
					errW = webhooks.ErrorAction("finished", err.Error(), "Failed to init cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return err
				}
				if promptGen == "" {
					log.Println("failed to generate prompt")
					errW = webhooks.ErrorAction("finished", "failed to generate prompt", "Failed to init cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return errors.New("failed to generate prompt")
				}
				payload.Prompt = promptGen
				err = SpinRequestConsumerApp(payload)
				if err != nil {
					const msg = "Cluster creation failed"
					errW = webhooks.ErrorAction("finished", msg, "failed to spin cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return err
				}

				return nil
			}

			errW = webhooks.ErrorAction("finished", "Unable to fetch any details from repository.", "Unable to fetch any details from repository, Failed to init cluster", false)
			log.Println("Unable to fetch any details from repository")
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return errors.New("Unable to fetch any details from repository")
			// take descfromvision, live url as preference, if nil then fallback to descfromscan

		}

		if response.Clone_type == "url" {
			err := scrapeAndCloneUrl(response.Url, payload, client)
			return err
		}
	}
	if err != nil {
		log.Println("error while spinning up request.")
		return err
	}
	log.Printf("🔧 Processing task: %s", string(body))
	return nil
}

func scrapeAndCloneUrl(url string, payload SpinRequest, client interfaces.LLMClient) error {
	errW := webhooks.PrecheckAction("finished", "Navigating "+url)
	if errW != nil {
		log.Println("api-service webhook failed")
	}
	// vision, extractor -> prompt+details -> code gen

	if url != "" {
		_, ok := helpers.IsClonableWebApp(url)
		if ok != "" {
			errW = webhooks.ErrorAction("finished", ok, "failed to spin cluster", false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return errors.New(ok)
		}

		errW = webhooks.PrecheckAction("finished", "checking if url is clonable")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		//getting vision details with check (if a website is eligible for clone) ----
		resp, err := headless.GenerateClonePromptByVision(url, client, false)
		if err != nil {
			errW = webhooks.ErrorAction("finished", err.Error(), "failed to spin cluster", false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
		if value, ok := resp["is_clonable"]; ok {
			if clonable, ok := value.(bool); ok && !clonable {
				reason, ok := resp["reason"].(string)
				if ok && reason != "" {
					errW = webhooks.ErrorAction("finished", reason, "failed to spin cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return errors.New(reason)
				} else {
					errW = webhooks.ErrorAction("finished", "failed to fetch failure reason", "failed to spin cluster", false)
					if errW != nil {
						log.Println("api-service webhook failed")
					}
					return errors.New("failed to fetch failure reason")
				}
			}
		}

		//no error found --
		// run image extractor ----
		errW = webhooks.PrecheckAction("finished", "proceeding with assets extraction")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		cloner, err := headless.NewWebsiteCloner("testdir")
		if err != nil {
			log.Println(err)
			fmt.Printf("Error creating cloner: %v\n", err)

			//calling error webhook --
			const msg = "Cluster creation failed"
			errW = webhooks.ErrorAction("finished", msg, err.Error(), false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			//
			return err
		}
		defer cloner.Close()
		errW = webhooks.PrecheckAction("finished", "Extracting Assets...")
		if errW != nil {
			log.Println("api-service webhook failed")
		}
		assets := []string{}
		extraction, err := cloner.CloneSite(url)
		if err != nil {
			log.Println("there was an error extracting images")
			//error webhook no images ---
			const msg = "Cluster creation failed"
			errW = webhooks.ErrorAction("finished", msg, err.Error(), false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
			//
		}
		assets = append(assets, extraction.Images...)

		resp["images"] = assets
		resp["title"] = extraction.Title
		layoutResponse := resp //map

		jsonBytes, err := json.Marshal(layoutResponse)
		if err != nil {
			//error webhook
			const msg = "Cluster creation failed"
			errW = webhooks.ErrorAction("finished", msg, err.Error(), false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			//
			return err
		}
		payload.Prompt = string(jsonBytes)
		payload.IsClone = true

		errW = webhooks.PrecheckAction("finished", "Collecting extracted assets...")
		if errW != nil {
			log.Println("api-service webhook failed")
		}

		log.Println("Proceeding to generate next js code.")

		err = SpinRequestConsumerApp(payload)
		if err != nil {
			const msg = "Cluster creation failed"
			errW = webhooks.ErrorAction("finished", msg, "failed to spin cluster", false)
			if errW != nil {
				log.Println("api-service webhook failed")
			}
			return err
		}
	}

	return nil
}
