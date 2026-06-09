package contract

type RuntimeEnvRequest struct {
	Project     string `json:"project"`
	Environment string `json:"environment"`
	Reason      string `json:"reason"`
}

type RuntimeEnvResponse struct {
	Project     string            `json:"project"`
	Environment string            `json:"environment"`
	Env         map[string]string `json:"env"`
}
