package consumers

type UpdateRequest struct {
	Cluster string
	Prompt  string
	Message string
	UserId  string
}

func updateRequestConsumer(spinRequest SpinRequest) error {

	return nil
}
