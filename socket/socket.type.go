package socket

// Socket struct
type Socket struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Mode         string `json:"mode,omitempty"` // Server | client
	Description  string `json:"description,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
	Host         string `json:"host,omitempty"`
	Port         string `json:"port,omitempty"`
	Event        Event  `json:"event,omitempty"`
	Timeout      int    `json:"timeout,omitempty"` // timeout (seconds)
	BufferSize   int    `json:"buffer,omitempty"`  // bufferSize
	KeepAlive    int    `json:"keep,omitempty"`    // -1 not keep alive, 0 keep alive always, keep alive n seconds.
	Process      string `json:"process,omitempty"`
	AttemptAfter int    `json:"attempt_after,omitempty"` // Attempt attempt_after
	Attempts     int    `json:"attempts,omitempty"`      // max times try to reconnect server when connection break (client mode only)
	client       *Client
}

// Event struct
type Event struct {
	Data      string `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Closed    string `json:"closed,omitempty"`
	Connected string `json:"connected,omitempty"`
}
