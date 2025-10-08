package main

import "os"

func (cfg *apiConfig) cleanupImages() {
	os.RemoveAll(cfg.tempRoot)
	os.MkdirAll(cfg.tempRoot, 0755)
}
