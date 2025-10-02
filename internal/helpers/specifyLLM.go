package helpers

import (
	"errors"
	"log"
	"strings"

	"github.com/substrate-cli/consumer-service-cli/internal/utils"
)

func SpecifyModel(modelName string) error {
	supportedModels := strings.Split(utils.GetSupportedModels(), ",")
	modelName = strings.ToLower(modelName)
	defaultModel := utils.GetDefaultModel()

	if modelName == "" || len(modelName) == 0 {
		utils.SetModel(defaultModel)
		return nil
	}

	isValid := isValidModel(modelName, supportedModels)

	if isValid {
		utils.SetModel(modelName)
	} else {
		log.Println("invalid model reference")
		// utils.SetModel(defaultModel)
		return errors.New("invalid model provided.")
	}
	return nil
}

func isValidModel(model string, list []string) bool {

	for _, m := range list {
		if model == m {
			return true
		}
	}

	return false
}
