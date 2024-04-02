package crawler

import (
	"mmf/config"
	"mmf/pkg/calculation"
	"mmf/pkg/constants"
)

func StartCrawler(config config.MMRConfig) bool {
	for _, queue := range constants.GetAllQueueTypes() {
		calculation.EvaluateTickets(config, queue)
	}
	return true
}
