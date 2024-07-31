package crawler

import (
	"mmf/config"
	"mmf/internal/calculation"
	"mmf/internal/constants"
)

func StartCrawler(config config.MMRConfig) bool {
	for _, queue := range constants.GetAllQueueTypes() {
		calculation.EvaluateTickets(config, queue)
	}
	return true
}
