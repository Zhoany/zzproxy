package main

import (
	"log"
	"zzproxy/config"
	"zzproxy/server"
)


func main() {
    // 1. 加载配置
    cfg, err := config.LoadConfig("cfg.yaml")
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
	  server.Run(cfg)  // 传入 cfg
    select {}
}
