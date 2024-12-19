package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("../../config/meta_server.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	assert.NotNil(t, config)
	assert.Equal(t, "meta-1", config.ServerID)
	assert.Equal(t, "meta", config.ServerType)
}
