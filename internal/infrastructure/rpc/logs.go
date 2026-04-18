package rpc

import "encoding/json"

// LogsNotification — мінімальний парсинг відповіді від Solana
type LogsNotification struct {
	Method string `json:"method"`
	Params struct {
		Result struct {
			Context struct {
				Slot uint64 `json:"slot"`
			} `json:"context"`
			Value struct {
				Signature string   `json:"signature"`
				Logs      []string `json:"logs"`
			} `json:"value"`
		} `json:"result"`
	} `json:"params"`
}

// ParseLogs — витягує signature + slot + logs
func ParseLogs(msg []byte) (*LogsNotification, error) {
	var n LogsNotification
	if err := json.Unmarshal(msg, &n); err != nil {
		return nil, err
	}

	if n.Method != "logsNotification" {
		return nil, nil
	}

	return &n, nil
}
