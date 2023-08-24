package transport

type TriggerParam struct {
	JobId                 int32  `json:"jobId"`                 // 任务ID
	ExecutorHandler       string `json:"executorHandler"`       // 任务标识
	ExecutorParams        string `json:"executorParams"`        // 任务参数
	ExecutorBlockStrategy string `json:"executorBlockStrategy"` // 任务阻塞策略，可选值参考 com.xxl.job.core.enums.ExecutorBlockStrategyEnum
	ExecutorTimeout       int32  `json:"executorTimeout"`       // 任务超时时间，单位秒，大于零时生效
	LogId                 int64  `json:"logId"`                 // 本次调度日志ID
	LogDateTime           int64  `json:"logDateTime"`           // 本次调度日志时间
	GlueType              string `json:"glueType"`              // 任务模式，可选值参考 com.xxl.job.core.glue.GlueTypeEnum
	GlueSource            string `json:"glueSource"`            // GLUE脚本代码
	GlueUpdatetime        int64  `json:"glueUpdatetime"`        // GLUE脚本更新时间，用于判定脚本是否变更以及是否需要刷新
	BroadcastIndex        int32  `json:"broadcastIndex"`        // 分片参数：当前分片
	BroadcastTotal        int32  `json:"broadcastTotal"`        // 分片参数：总分片
}

type ReturnT struct {
	Code    int32       `json:"code"`
	Msg     string      `json:"msg"`
	Content interface{} `json:"content"`
}

type HandleCallbackParam struct {
	LogId      int64  `json:"logId"`
	LogDateTim int64  `json:"logDateTim"`
	Code       int32  `json:"handleCode"`
	Msg        string `json:"handleMsg"`
}

type RegistryParam struct {
	RegistryGroup string `json:"registryGroup"`
	RegistryKey   string `json:"registryKey"`
	RegistryValue string `json:"registryValue"`
}

type LogRequest struct {
	LogDateTim  int64 `json:"logDateTim"`
	LogId       int64 `json:"logId"`
	FromLineNum int32 `json:"fromLineNum"`
}
