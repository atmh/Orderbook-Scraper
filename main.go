package main

import (
	"context"
	"flag"
	"log"
	"sync"

	"OrderBookScraper/api"
)

func main() {
	currency := flag.String("currency", "BTC", "Specify the currency (default: BTC)")
	flag.Parse()

	// Fetch all options contracts from Deribit
	contracts, err := api.GetAllDeribitOptionContracts(*currency)
	if err != nil {
		log.Fatal("Failed to fetch contracts:", err)
		return
	}

	log.Printf("Found %d contracts, subscribing using multiple WebSockets...", len(contracts))

	// Use multiple WebSockets in batches, with each batch having its own producer to publish to the kafka topic.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < len(contracts); i += api.OptionsContractBatchSize {
		end := i + api.OptionsContractBatchSize
		if end > len(contracts) {
			end = len(contracts)
		}
		wg.Add(1)
		go api.BatchSubscribeDeribitContracts(ctx, &wg, contracts[i:end])
	}
	wg.Wait()
}
