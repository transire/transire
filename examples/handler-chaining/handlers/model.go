package handlers

const (
	WorkQueue    = "work"
	SummaryQueue = "summary-log"
	LogQueue     = "log-stream"
)

type WorkPayload struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type SummaryPayload struct {
	Source string   `json:"source"`
	Steps  []string `json:"steps"`
}

type LogPayload struct {
	Message string `json:"message"`
}
