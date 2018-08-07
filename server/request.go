package server

type Request interface {
	Response(req *AgentRequest) error
}
