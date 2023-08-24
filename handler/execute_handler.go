package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gongshen/xxl-job-client/constants"
	"github.com/gongshen/xxl-job-client/logger"
	"github.com/gongshen/xxl-job-client/transport"
)

var scriptMap = map[string]string{
	"GLUE_SHELL":      ".sh",
	"GLUE_PYTHON":     ".py",
	"GLUE_PHP":        ".php",
	"GLUE_NODEJS":     ".js",
	"GLUE_POWERSHELL": ".ps1",
}

var scriptCmd = map[string]string{
	"GLUE_SHELL":      "bash",
	"GLUE_PYTHON":     "python3",
	"GLUE_PHP":        "php",
	"GLUE_NODEJS":     "node",
	"GLUE_POWERSHELL": "powershell",
}

type ExecuteHandler interface {
	ParseJob(trigger *transport.TriggerParam) (runParam *JobRunParam, err error)
	Execute(jobId int32, glueType string, runParam *JobRunParam) error
}

type ScriptHandler struct {
	sync.RWMutex
}

// ParseJob 根据trigger参数获取JobRun参数
func (s *ScriptHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	suffix, ok := scriptMap[trigger.GlueType]
	if !ok {
		logParam := make(map[string]interface{})
		logParam["logId"] = trigger.LogId
		logParam["jobId"] = trigger.JobId

		jobParamMap := make(map[string]map[string]interface{})
		jobParamMap["logParam"] = logParam
		ctx := context.WithValue(context.Background(), "jobParam", jobParamMap)

		msg := "暂不支持" + strings.ToLower(trigger.GlueType[constants.GluePrefixLen:]) + "脚本"
		logger.Info(ctx, "job parse error:", msg)
		return jobParam, errors.New(msg)
	}

	path := fmt.Sprintf("%s%d_%d%s", constants.GlueSourcePath, trigger.JobId, trigger.GlueUpdatetime, suffix)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		log.Printf("script file not exist,need create. jobId:%d,content:%s\n", trigger.JobId, trigger.GlueSource)
		s.Lock()
		defer s.Unlock()
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0750)
		if err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(constants.GlueSourcePath, os.ModePerm)
			if err == nil {
				file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0750)
				if err != nil {
					return jobParam, err
				}
			}
		}

		if file != nil {
			defer file.Close()
			res, err := file.Write([]byte(trigger.GlueSource))
			if err != nil {
				return jobParam, err
			}
			if res <= 0 {
				return jobParam, errors.New("write script file failed")
			}
		}
	}

	inputParam := make(map[string]interface{})
	if trigger.ExecutorParams != "" {
		inputParam["param"] = trigger.ExecutorParams
	}

	jobParam = &JobRunParam{
		LogId:                 trigger.LogId,
		LogDateTime:           trigger.LogDateTime,
		JobName:               trigger.ExecutorHandler,
		JobTag:                path,
		InputParam:            inputParam,
		ExecutorBlockStrategy: trigger.ExecutorBlockStrategy,
	}
	if trigger.BroadcastTotal > 0 {
		jobParam.ShardIdx = trigger.BroadcastIndex
		jobParam.ShardTotal = trigger.BroadcastTotal
	}
	return jobParam, nil
}

func (s *ScriptHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logParam := make(map[string]interface{})
	logParam["logId"] = runParam.LogId
	logParam["jobId"] = jobId
	logParam["jobName"] = runParam.JobName
	logParam["jobFunc"] = runParam.JobTag

	shardParam := make(map[string]interface{})
	shardParam["shardingIdx"] = runParam.ShardIdx
	shardParam["shardingTotal"] = runParam.ShardTotal

	jobParam := make(map[string]map[string]interface{})
	jobParam["logParam"] = logParam
	jobParam["inputParam"] = runParam.InputParam
	jobParam["sharding"] = shardParam
	ctx := context.WithValue(context.Background(), "jobParam", jobParam)

	basePath := logger.GetLogPath(time.Now())
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		s.Lock()
		os.MkdirAll(basePath, os.ModePerm)
		s.Unlock()
	}
	logPath := basePath + fmt.Sprintf("/%d", runParam.LogId) + ".log"
	args := make([]string, 0)
	//args = append(args, "-c")
	// 放入脚本位置参数
	args = append(args, runParam.JobTag)
	// 执行参数
	var parameters string
	if len(runParam.InputParam) > 0 {
		ps, ok := runParam.InputParam["param"]
		if ok {
			parameters = ps.(string)
		}
	}
	args = append(args, parameters)
	args = append(args, strconv.Itoa(int(runParam.ShardIdx)))
	args = append(args, strconv.Itoa(int(runParam.ShardTotal)))

	cancelCtx, canFun := context.WithCancel(context.Background())
	defer canFun()

	runParam.CurrentCancelFunc = canFun
	cmd := exec.CommandContext(cancelCtx, scriptCmd[glueType], args...)
	logger.Info(ctx, fmt.Sprintf("Script Execute. jobId:%d,logPath:%s,cmd:%s", jobId, logPath, strings.Join(args, " ")))
	// 文件需要append属性
	f, _ := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	defer f.Close()
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Run(); err != nil {
		logger.Info(ctx, "run script job err:", err)
		return err
	}
	return nil
}

type BeanHandler struct {
	RunFunc JobHandlerFunc
}

func (b *BeanHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	if b.RunFunc == nil {
		return jobParam, errors.New("job run function not found")
	}

	inputParam := make(map[string]interface{})
	if trigger.ExecutorParams != "" {
		params := strings.Split(trigger.ExecutorParams, ",")
		if len(params) > 0 {
			for _, param := range params {
				if param != "" {
					jobP := strings.Split(param, "=")
					if len(jobP) > 1 {
						inputParam[jobP[0]] = jobP[1]
					}
				}
			}
		}
	}

	funName := getFunctionName(b.RunFunc)
	jobParam = &JobRunParam{
		LogId:                 trigger.LogId,
		LogDateTime:           trigger.LogDateTime,
		JobName:               trigger.ExecutorHandler,
		JobTag:                funName,
		InputParam:            inputParam,
		ExecutorBlockStrategy: trigger.ExecutorBlockStrategy,
	}
	return jobParam, err
}

func (b *BeanHandler) Execute(jobId int32, _ string, runParam *JobRunParam) (err error) {
	logParam := make(map[string]interface{})
	logParam["logId"] = runParam.LogId
	logParam["jobId"] = jobId
	logParam["jobName"] = runParam.JobName
	logParam["jobFunc"] = runParam.JobTag

	shardParam := make(map[string]interface{})
	shardParam["shardingIdx"] = runParam.ShardIdx
	shardParam["shardingTotal"] = runParam.ShardTotal

	jobParam := make(map[string]map[string]interface{})
	jobParam["logParam"] = logParam
	jobParam["inputParam"] = runParam.InputParam
	jobParam["sharding"] = shardParam

	valueCtx, canFun := context.WithCancel(context.Background())
	defer canFun()

	runParam.CurrentCancelFunc = canFun
	ctx := context.WithValue(valueCtx, "jobParam", jobParam)
	// add recover
	defer func() {
		r := recover()
		if r != nil {
			err = errors.New(fmt.Sprintf("panic:%v", r))
			logger.Info(ctx, "job run failed! msg:", err.Error())
		}
	}()

	err = b.RunFunc(ctx)
	if err != nil {
		logger.Info(ctx, "job run failed! msg:", err.Error())
		return
	}

	return
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
