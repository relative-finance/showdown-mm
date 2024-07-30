package constants

type QueueType string

const (
	CS2Queue QueueType = "cs2queue"
	D2Queue  QueueType = "d2queue"
	LCQueue  QueueType = "lcqueue"
)

func GetQueueType(queue string) QueueType {
	switch queue {
	case "cs2queue":
		return CS2Queue
	case "d2queue":
		return D2Queue
	case "lcqueue":
		return LCQueue
	default:
		return ""
	}
}

func (g *QueueType) String() string {
	return string(*g)
}

func GetAllQueueTypes() []QueueType {
	return []QueueType{CS2Queue, D2Queue, LCQueue}
}

func GetIndexNameQueue(queue QueueType) string {
	return "players_" + queue.String()
}

func GetIndexNameStr(queue string) string {
	return "players_" + queue
}
