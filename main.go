package main

import (
	"github.com/feixiaobo/go-xxl-job-client/v2"
	"github.com/feixiaobo/go-xxl-job-client/v2/option"
)

func main() {
	client := xxl.NewXxlClient(
		option.WithAppName("test"),
		option.WithClientPort(8088),
		option.WithAdminAddress("http://172.16.243.253:8088/xxl-job-admin"),
		option.WithEnableHttp(true),
	)
	client.Run()
}
