package handler

import (
	"encoding/json"
	"github.com/gongshen/xxl-job-client/logger"
	"github.com/gongshen/xxl-job-client/transport"
	"log"
)

type HttpRequestHandler struct{}

func (h HttpRequestHandler) Beat() error {
	return nil
}

func (h HttpRequestHandler) IdleBeat(body []byte) (jobId int32, err error) {
	job := &JobId{}
	err = json.Unmarshal(body, job)
	if err != nil {
		return 0, err
	}
	return job.JobId, err
}

func (h HttpRequestHandler) Run(body []byte) (triggerParam *transport.TriggerParam, err error) {
	err = json.Unmarshal(body, &triggerParam)
	if err != nil {
		log.Printf("HttpRequestHandler Run Err: body:%s\n", string(body))
	}
	return triggerParam, err
}

type JobId struct {
	JobId int32 `json:"jobId"`
}

func (h HttpRequestHandler) Kill(body []byte) (jobId int32, err error) {
	job := &JobId{}
	err = json.Unmarshal(body, job)
	if err != nil {
		return 0, err
	}
	return job.JobId, err
}

func (h HttpRequestHandler) Log(body []byte) (log *logger.LogResult, err error) {
	lq := &transport.LogRequest{}
	err = json.Unmarshal(body, lq)
	if err != nil {
		return nil, err
	}
	line, content := logger.ReadLog(lq.LogDateTim, lq.LogId, lq.FromLineNum)
	log = &logger.LogResult{
		FromLineNum: lq.FromLineNum,
		ToLineNum:   line,
		LogContent:  content,
		IsEnd:       true,
	}
	return log, err
}
