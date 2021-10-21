package main

import (
	"github.com/feixiaobo/go-xxl-job-client/v2"
	"github.com/feixiaobo/go-xxl-job-client/v2/option"
	"github.com/jinzhu/configor"
	"log"
)

type Config struct {
	AppName      string `yaml:"app_name"`
	ClientPort   int    `yaml:"client_port"`
	AdminAddress string `yaml:"admin_address"`
}

func main() {
	cnf := &Config{}
	if err := configor.Load(cnf, "config.yaml"); err != nil {
		log.Fatalf("failed to load local config file:%v", err)
	}
	client := xxl.NewXxlClient(
		option.WithAppName(cnf.AppName),
		option.WithClientPort(cnf.ClientPort),
		option.WithAdminAddress(cnf.AdminAddress),
		option.WithEnableHttp(true),
	)
	client.Run()
}
