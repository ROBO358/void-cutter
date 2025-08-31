package config

import "fmt"

// Config holds all configuration parameters for void-cutter
type Config struct {
	// Input files
	InputFiles []string

	// Output settings
	OutputSuffix string

	// Loudness normalization settings
	TargetLoudness float64 // LUFS

	// Silence detection settings
	SilenceThreshold    float64 // dBFS
	MinSilenceDuration  int     // milliseconds
	KeepSilenceDuration int     // milliseconds
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		OutputSuffix:        "_edited",
		TargetLoudness:      -16.0,
		SilenceThreshold:    -50.0,
		MinSilenceDuration:  500,
		KeepSilenceDuration: 250,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.InputFiles) == 0 {
		return fmt.Errorf("no input files specified")
	}

	if c.SilenceThreshold < -120.0 || c.SilenceThreshold > 0.0 {
		return fmt.Errorf("silence threshold must be between -120.0 and 0.0 dBFS")
	}

	if c.MinSilenceDuration <= 0 {
		return fmt.Errorf("minimum silence duration must be positive")
	}

	if c.KeepSilenceDuration < 0 {
		return fmt.Errorf("keep silence duration must be non-negative")
	}

	return nil
}
