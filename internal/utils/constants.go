package utils

type requestSpin struct {
	serverStructCall string
	serverGenCall    string
	appGenFSCall     string
	appGenCall       string
}

type constantsConfig struct {
	requestSpin requestSpin
}

var constants *constantsConfig

func init() {
	constants = &constantsConfig{
		requestSpin: requestSpin{
			serverStructCall: "spin.generateServerStruct.llmrequest",
			serverGenCall:    "spin.generateServerCode.llmrequest",
			appGenFSCall:     "spin.generateAppFSCode.llmrequest",
			appGenCall:       "spin.generateAppCode.llmrequest",
		},
	}
}

func GetServerStructCall() *string {
	return &constants.requestSpin.serverStructCall
}

func GetServerGenCall() *string {
	return &constants.requestSpin.serverGenCall
}

func GetAppGenFSCall() *string {
	return &constants.requestSpin.appGenFSCall
}

func GetAppGenCall() *string {
	return &constants.requestSpin.appGenCall
}
