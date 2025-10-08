package main

import (
	"os"
)

func (cfg apiConfig) ensureDirs() error {
	err := cfg.ensureAssetsDir()
	if err != nil {
		return err
	}
	return cfg.ensureTempDir()
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) ensureTempDir() error {
	if _, err := os.Stat(cfg.tempRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.tempRoot, 0755)
	}
	return nil
}
