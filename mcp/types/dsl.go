package types

import (
	"fmt"
	"time"
)

// GetEnvs get the environment variables
func (client *ClientDSL) GetEnvs() []string {
	if client.Env == nil {
		return []string{}
	}

	envs := []string{}
	for k, v := range client.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}

// GetTimeout get the timeout
func (client *ClientDSL) GetTimeout() time.Duration {
	if client.Timeout == "" {
		return 0
	}

	// Use time.ParseDuration for standard formats (1s, 5m, 1h, etc.)
	duration, err := time.ParseDuration(client.Timeout)
	if err != nil {
		return 0
	}

	return duration
}

// GetAuthorizationToken get the authorization token
func (client *ClientDSL) GetAuthorizationToken() string {
	return client.AuthorizationToken
}

// GetVersion get the version, default to "1.0.0" if not set
func (client *ClientDSL) GetVersion() string {
	if client.Version == "" {
		return "1.0.0"
	}
	return client.Version
}

// GetImplementation get the implementation info
func (client *ClientDSL) GetImplementation() Implementation {
	return Implementation{
		Name:    client.Name,
		Version: client.GetVersion(),
	}
}

// GetClientCapabilities get the default client capabilities
func (client *ClientDSL) GetClientCapabilities() ClientCapabilities {
	caps := ClientCapabilities{
		Experimental: make(map[string]interface{}),
	}

	// Enable sampling capability if configured
	if client.EnableSampling {
		caps.Sampling = &SamplingCapability{}
	}

	// Enable roots capability if configured
	if client.EnableRoots {
		caps.Roots = &RootsCapability{
			ListChanged: client.RootsListChanged,
		}
	}

	// Enable elicitation capability if configured
	if client.EnableElicitation {
		caps.Elicitation = &ElicitationCapability{}
	}

	return caps
}
