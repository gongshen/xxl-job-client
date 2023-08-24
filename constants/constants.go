package constants

const (
	DateTimeFormat = "2006-01-02 15:04:05"
	DateFormat     = "2006-01-02"
	BasePath       = "/data/applogs/xxl-job/jobhandler/"
	GlueSourcePath = BasePath + "gluesource/"
	GluePrefix     = "GLUE_"
	GluePrefixLen  = len(GluePrefix)
)

// 阻塞处理策略
const (
	SerialExecution = "SERIAL_EXECUTION" //单机串行
	DiscardLater    = "DISCARD_LATER"    //丢弃后续调度
	CoverEarly      = "COVER_EARLY"      //覆盖之前调度
)
