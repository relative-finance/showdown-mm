package crawler

import (
	"log"
	"mmf/pkg/calculation"
)

func StartCrawler(mode string) bool {

	calculation.EvaluateTickets(mode)
	log.Printf("bruh")
	return true
}
