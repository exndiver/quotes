package fuel

import (
	"encoding/json"
	"os"
)

// FuelConfig holds the configuration for fuel price collection
type FuelConfig struct {
	Sources        map[string]SourceConfig `json:"sources"`
	Countries      []CountryConfig         `json:"countries"`
	UpdateInterval string                  `json:"update_interval"`
}

// SourceConfig defines a price source status and details
type SourceConfig struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// CountryConfig links a country to a source and status
type CountryConfig struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
}

// LoadConfig reads the fuel configuration from a JSON file
func LoadConfig(path string) (*FuelConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config FuelConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
