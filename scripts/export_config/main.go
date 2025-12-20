/*
Export internal/config/config.go configuration to YAML file.
*/
package main

import (
	"fmt"
	"os"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"gopkg.in/yaml.v3"
)

func main() {
	// Initialize configuration
	cfg := config.NewConfig()

	// Marshal configuration to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Printf("Error marshaling config to YAML: %v\n", err)
		return
	}

	// Write YAML data to file
	filePath := "yqlx-config.yaml"
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		fmt.Printf("Error writing config to file: %v\n", err)
		return
	}

	fmt.Printf("Configuration exported successfully to %s\n", filePath)
}
