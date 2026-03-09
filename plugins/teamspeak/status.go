package teamspeak

import "time"

type Status struct {
	IsMessagePinned bool

	ResetTicker (chan bool)

	IsInRetryLoop bool

	RetryCount       int
	CheckCount       int
	CheckFailedCount int

	BeforeOnlineClient []OnlineClient

	IsCheckClientTaskScheduled bool
	IsCheckClientTaskRunning   bool

	IsDeleteMessageTaskScheduled bool
	IsDeleteMessageTaskRunning   bool

	OldMessageID []OldMessageID

	RetryMsgID int
}

type OldMessageID struct {
	Date int
	ID   int
}

type OnlineClient struct {
	Username   string
	DatabaseID int
	JoinTime   time.Time
}
