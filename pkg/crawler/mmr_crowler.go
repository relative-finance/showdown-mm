package crawler

import (
	"log"
	"mmf/config"
	"mmf/pkg/calculation"
)

func StartCrawler(config config.MMRConfig) bool {

	calculation.EvaluateTickets(config)
	log.Printf("Crawlin'...")
	return true
}
