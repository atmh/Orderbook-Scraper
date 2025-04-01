package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"OrderBookScraper/kafka"
)

const (
	deribitWS                = "wss://test.deribit.com/ws/api/v2"
	deribitGetInstruments    = "https://test.deribit.com/api/v2/public/get_instruments?currency=%s&kind=option"
	OptionsContractBatchSize = 50 // Maximum number of contracts per WebSocket subscription batch.
)

// Request structure for Deribit WebSocket API.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Instrument represents a contract fetched from the Deribit API.
type Instrument struct {
	InstrumentName string `json:"instrument_name"`
}

// InstrumentsResponse Structure for the API response containing multiple instruments.
type InstrumentsResponse struct {
	Result []Instrument `json:"result"`
}

// GetAllDeribitOptionContracts Fetches all BTC options contracts from Deribit
func GetAllDeribitOptionContracts(currency string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf(deribitGetInstruments, currency))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result InstrumentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var contracts []string
	for _, instr := range result.Result {
		contracts = append(contracts, instr.InstrumentName)
	}

	return contracts, nil
}

// BatchSubscribeDeribitContracts Handles subscription to WebSocket channels for a batch of BTC options contracts.
// Updates from the WebSocket are produced to a Kafka topic.
func BatchSubscribeDeribitContracts(ctx context.Context, wg *sync.WaitGroup, contracts []string) {
	defer wg.Done()

	// Initialize Kafka producer for this batch
	producer, err := kafka.InitKafkaProducer()
	if err != nil {
		log.Println("[BatchSubscribeDeribitContracts] Failed to initialize Kafka producer for batch:", err)
		return
	}
	defer producer.Close()

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(deribitWS, nil)
	if err != nil {
		log.Println("[BatchSubscribeDeribitContracts] WebSocket connection error:", err)
		return
	}
	defer conn.Close()

	// Subscribe to batch of contracts
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "public/subscribe",
		Params: map[string]interface{}{
			"channels": createOrderBookChannels(contracts),
		},
	}

	if err := conn.WriteJSON(req); err != nil {
		log.Println("[BatchSubscribeDeribitContracts] Subscription error:", err)
		return
	}

	// Listen for order book updates and produce to Kafka
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("[BatchSubscribeDeribitContracts] Read error:", err)
				// Attempt reconnection on failure
				handleReconnection(ctx, contracts, wg)
				return
			}

			// Send the message to Kafka
			err = kafka.ProduceToKafka(producer, msg)
			if err != nil {
				log.Println("[BatchSubscribeDeribitContracts] Error producing to Kafka:", err)
				// Retry sending the message if Kafka fails
				kafka.RetryProduceToKafka(producer, msg)
			}
		}
	}
}

// createOrderBookChannels Creates an array of Deribit WebSocket channels for order book updates.
func createOrderBookChannels(contracts []string) []string {
	var channels []string
	for _, contract := range contracts {
		channels = append(channels, "book."+contract+".100ms")
	}
	return channels
}

// handleReconnection Handles reconnection logic for WebSocket subscriptions in case of errors.
func handleReconnection(ctx context.Context, contracts []string, wg *sync.WaitGroup) {
	log.Println("[handleReconnection] Attempting to reconnect...")
	time.Sleep(5 * time.Second) // Exponential backoff could be applied here
	select {
	case <-ctx.Done():
		return
	default:
		// Attempt to re-subscribe and re-establish connection
		wg.Add(1)
		BatchSubscribeDeribitContracts(ctx, wg, contracts)
	}
}
