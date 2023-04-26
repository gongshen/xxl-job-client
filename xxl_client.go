package xxl

import (
	"context"
	"github.com/gongshen/xxl-job-client/admin"
	executor2 "github.com/gongshen/xxl-job-client/executor"
	"github.com/gongshen/xxl-job-client/handler"
	"github.com/gongshen/xxl-job-client/logger"
	"github.com/gongshen/xxl-job-client/option"
)

type XxlClient struct {
	executor       *executor2.Executor
	requestHandler *handler.RequestProcess
}

func NewXxlClient(opts ...option.Option) *XxlClient {
	clientOps := option.NewClientOptions(opts...)
	executor := executor2.NewExecutor(
		clientOps.AppName,
		clientOps.Port,
	)

	adminServer := admin.NewAdminServer(
		clientOps.AdminAddr,
		clientOps.Timeout,
		clientOps.BeatTime,
		executor,
	)

	var requestHandler *handler.RequestProcess
	adminServer.AccessToken = map[string]string{
		"XXL-JOB-ACCESS-TOKEN": clientOps.AccessToken,
	}

	requestHandler = handler.NewRequestProcess(adminServer, &handler.HttpRequestHandler{})
	httpServer := executor2.NewHttpServer(requestHandler.RequestProcess)
	executor.SetServer(httpServer)

	return &XxlClient{
		requestHandler: requestHandler,
		executor:       executor,
	}
}

func (c *XxlClient) ExitApplication() {
	c.requestHandler.RemoveRegisterExecutor()
}

func GetParam(ctx context.Context, key string) (val string, has bool) {
	jobMap := ctx.Value("jobParam")
	if jobMap != nil {
		inputParam, ok := jobMap.(map[string]map[string]interface{})["inputParam"]
		if ok {
			val, vok := inputParam[key]
			if vok {
				return val.(string), true
			}
		}
	}
	return "", false
}

func GetSharding(ctx context.Context) (shardingIdx, shardingTotal int32) {
	jobMap := ctx.Value("jobParam")
	if jobMap != nil {
		shardingParam, ok := jobMap.(map[string]map[string]interface{})["sharding"]
		if ok {
			idx, vok := shardingParam["shardingIdx"]
			if vok {
				shardingIdx = idx.(int32)
			}
			total, ok := shardingParam["shardingTotal"]
			if ok {
				shardingTotal = total.(int32)
			}
		}
	}
	return shardingIdx, shardingTotal
}

func (c *XxlClient) Run() error {
	c.requestHandler.RegisterExecutor()
	logger.InitLogPath()
	return c.executor.Run()
}

func (c *XxlClient) Close() error {
	if err := c.executor.Close(); err != nil {
		return err
	}
	return nil
}

func (c *XxlClient) RegisterJob(jobName string, function handler.JobHandlerFunc) {
	c.requestHandler.RegisterJob(jobName, function)
}
