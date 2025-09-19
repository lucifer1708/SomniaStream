package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/rs/cors"
)

// Config holds the configuration for the devtool
type Config struct {
	RPCEndpoint string
	NATSUrl     string
	NATSToken   string
	ServerPort  string
}

// DevTool represents the main application
type SomniaStream struct {
	config    *Config
	rpcClient *rpc.Client
	ethClient *ethclient.Client
	natsConn  *nats.Conn
	js        nats.JetStreamContext
	upgrader  websocket.Upgrader
	router    *gin.Engine
}

// NewDevTool creates a new DevTool instance
func NewSomniaStream(config *Config) (*SomniaStream, error) {
	// Connect to RPC
	rpcClient, err := rpc.Dial(config.RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %v", err)
	}

	// Connect to Ethereum client
	ethClient, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	// Connect to NATS
	natsConn, err := nats.Connect(config.NATSUrl, nats.Name("devtool"), nats.Token(config.NATSToken))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}

	// Create JetStream context
	js, err := natsConn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %v", err)
	}

	// Initialize WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Setup CORS
	_ = cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	devtool := &SomniaStream{
		config:    config,
		rpcClient: rpcClient,
		ethClient: ethClient,
		natsConn:  natsConn,
		js:        js,
		upgrader:  upgrader,
		router:    router,
	}

	// Setup JetStream streams
	if err := devtool.setupJetStreams(); err != nil {
		return nil, fmt.Errorf("failed to setup JetStreams: %v", err)
	}

	return devtool, nil
}

// setupJetStreams creates the necessary JetStream streams
func (dt *SomniaStream) setupJetStreams() error {
	log.Println("Setting up JetStream streams...")

	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name:     "ETH_BLOCKS",
			subjects: []string{"eth.blocks.full", "eth.blocks"},
		},
		{
			name:     "ETH_TRANSACTIONS",
			subjects: []string{"eth.pending"},
		},
		{
			name:     "ETH_LOGS",
			subjects: []string{"eth.logs"},
		},
		{
			name:     "ETH_NETWORK",
			subjects: []string{"eth.network", "eth.gasPrice"},
		},
	}

	for _, stream := range streams {
		streamConfig := &nats.StreamConfig{
			Name:      stream.name,
			Subjects:  stream.subjects,
			Storage:   nats.MemoryStorage,
			Retention: nats.LimitsPolicy,
			MaxAge:    time.Hour * 24, // Keep data for 24 hours
			MaxMsgs:   10000,          // Keep up to 10k messages
		}

		// Try to get existing stream info first
		_, err := dt.js.StreamInfo(stream.name)
		if err != nil {
			// Stream doesn't exist, create it
			_, err = dt.js.AddStream(streamConfig)
			if err != nil {
				log.Printf("Failed to create stream %s: %v", stream.name, err)
				return err
			}
			log.Printf("Created JetStream stream: %s", stream.name)
		} else {
			log.Printf("JetStream stream already exists: %s", stream.name)
		}
	}

	log.Println("✅ JetStream streams setup complete")
	return nil
}

// Start starts the devtool server and RPC monitoring
func (dt *SomniaStream) Start(ctx context.Context) error {
	// Setup routes
	// dt.router.GET("/ws/:stream", dt.handleWebSocketStream)
	dt.router.GET("/sse/:stream", dt.handleSSEStream)
	dt.router.GET("/streams", dt.listStreams)
	dt.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start RPC monitoring
	go dt.monitorRPC(ctx)

	log.Printf("Starting server on port %s", dt.config.ServerPort)
	return dt.router.Run(":" + dt.config.ServerPort)
}

func (dt *SomniaStream) monitorRPC(ctx context.Context) {
	log.Println("Starting comprehensive RPC monitoring...")

	// Start multiple monitoring goroutines for different data types
	go dt.monitorBlocks(ctx)
	go dt.monitorPendingTransactions(ctx)
	go dt.monitorLogs(ctx)
	go dt.monitorNetworkStats(ctx)
	go dt.monitorGasPrice(ctx)

	// Keep the main monitoring goroutine alive
	<-ctx.Done()
	log.Println("RPC monitoring stopped")
}

// Monitor new blocks
func (dt *SomniaStream) monitorBlocks(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastBlockNumber uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dt.publishLatestBlock(&lastBlockNumber); err != nil {
				log.Printf("Error publishing block data: %v", err)
			}
		}
	}
}

// Monitor pending transactions
func (dt *SomniaStream) monitorPendingTransactions(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dt.publishPendingTransactions(); err != nil {
				log.Printf("Error publishing pending transactions: %v", err)
			}
		}
	}
}

// Monitor logs (events)
func (dt *SomniaStream) monitorLogs(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dt.publishRecentLogs(); err != nil {
				log.Printf("Error publishing logs: %v", err)
			}
		}
	}
}

// Monitor network statistics
func (dt *SomniaStream) monitorNetworkStats(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dt.publishNetworkStats(); err != nil {
				log.Printf("Error publishing network stats: %v", err)
			}
		}
	}
}

// Monitor gas price
func (dt *SomniaStream) monitorGasPrice(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dt.publishGasPrice(); err != nil {
				log.Printf("Error publishing gas price: %v", err)
			}
		}
	}
}

// Publish latest block with transaction details
func (dt *SomniaStream) publishLatestBlock(lastBlockNumber *uint64) error {
	log.Printf("[BLOCKS] Fetching latest block from Somnia RPC...")
	block, err := dt.ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Printf("[BLOCKS] ERROR: Failed to fetch latest block: %v", err)
		return err
	}

	currentBlockNumber := block.Number().Uint64()
	log.Printf("[BLOCKS] Current block number: %d, Last processed: %d", currentBlockNumber, *lastBlockNumber)

	if currentBlockNumber <= *lastBlockNumber {
		log.Printf("[BLOCKS] No new block, skipping...")
		return nil // No new block
	}
	*lastBlockNumber = currentBlockNumber

	log.Printf("[BLOCKS] Processing new block #%d with hash %s", currentBlockNumber, block.Hash().Hex())

	// Get block with full transactions
	blockWithTxs, err := dt.ethClient.BlockByNumber(context.Background(), block.Number())
	if err != nil {
		log.Printf("[BLOCKS] ERROR: Failed to fetch block with transactions: %v", err)
		return err
	}

	transactions := make([]map[string]interface{}, len(blockWithTxs.Transactions()))
	log.Printf("[BLOCKS] Block contains %d transactions", len(blockWithTxs.Transactions()))

	for i, tx := range blockWithTxs.Transactions() {
		transactions[i] = map[string]interface{}{
			"hash":     tx.Hash().Hex(),
			"to":       tx.To(),
			"value":    tx.Value().String(),
			"gasPrice": tx.GasPrice().String(),
			"gas":      tx.Gas(),
			"nonce":    tx.Nonce(),
		}
	}

	blockData := map[string]interface{}{
		"number":       block.Number().String(),
		"hash":         block.Hash().Hex(),
		"parentHash":   block.ParentHash().Hex(),
		"timestamp":    block.Time(),
		"gasUsed":      block.GasUsed(),
		"gasLimit":     block.GasLimit(),
		"difficulty":   block.Difficulty().String(),
		"size":         block.Size(),
		"txCount":      len(transactions),
		"transactions": transactions,
	}

	data, _ := json.Marshal(blockData)
	log.Printf("[BLOCKS] Publishing block data to JetStream (size: %d bytes)", len(data))

	_, err = dt.js.Publish("eth.blocks.full", data)
	if err != nil {
		log.Printf("[BLOCKS] ERROR: Failed to publish to JetStream: %v", err)
		return err
	}

	log.Printf("[BLOCKS] ✅ Successfully published block #%d to JetStream", currentBlockNumber)
	return nil
}

// Publish pending transactions
func (dt *SomniaStream) publishPendingTransactions() error {
	log.Printf("[PENDING] Fetching pending transactions from Somnia RPC...")
	var pendingTxs []map[string]interface{}

	// Get pending transactions using RPC call
	err := dt.rpcClient.Call(&pendingTxs, "eth_pendingTransactions")
	if err != nil {
		log.Printf("[PENDING] ERROR: Failed to fetch pending transactions: %v", err)
		return err
	}

	log.Printf("[PENDING] Found %d pending transactions", len(pendingTxs))

	if len(pendingTxs) > 0 {
		limitedTxs := pendingTxs[:min(len(pendingTxs), 50)] // Limit to 50 for performance
		data, _ := json.Marshal(map[string]interface{}{
			"count":        len(pendingTxs),
			"transactions": limitedTxs,
			"timestamp":    time.Now().Unix(),
		})

		log.Printf("[PENDING] Publishing %d pending transactions to JetStream (limited from %d)", len(limitedTxs), len(pendingTxs))

		_, err = dt.js.Publish("eth.pending", data)
		if err != nil {
			log.Printf("[PENDING] ERROR: Failed to publish to JetStream: %v", err)
			return err
		}

		log.Printf("[PENDING] ✅ Successfully published pending transactions to JetStream")
	} else {
		log.Printf("[PENDING] No pending transactions found")
	}

	return nil
}

// Publish recent logs
func (dt *SomniaStream) publishRecentLogs() error {
	// Get latest block number
	latestBlock, err := dt.ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}

	// Get logs from last 5 blocks
	fromBlock := latestBlock.Number().Uint64() - 5
	if fromBlock < 0 {
		fromBlock = 0
	}

	var logs []map[string]interface{}
	err = dt.rpcClient.Call(&logs, "eth_getLogs", map[string]interface{}{
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   "latest",
	})
	if err != nil {
		return err
	}

	if len(logs) > 0 {
		data, _ := json.Marshal(map[string]interface{}{
			"count":     len(logs),
			"logs":      logs[:min(len(logs), 100)], // Limit to 100 for performance
			"fromBlock": fromBlock,
			"toBlock":   latestBlock.Number().Uint64(),
			"timestamp": time.Now().Unix(),
		})
		_, err = dt.js.Publish("eth.logs", data)
		return err
	}

	return nil
}

// Publish network statistics
func (dt *SomniaStream) publishNetworkStats() error {
	var chainId, blockNumber, gasPrice, peerCount string

	// Get various network stats
	dt.rpcClient.Call(&chainId, "eth_chainId")
	dt.rpcClient.Call(&blockNumber, "eth_blockNumber")
	dt.rpcClient.Call(&gasPrice, "eth_gasPrice")
	dt.rpcClient.Call(&peerCount, "net_peerCount")

	var syncing interface{}
	dt.rpcClient.Call(&syncing, "eth_syncing")

	stats := map[string]interface{}{
		"chainId":     chainId,
		"blockNumber": blockNumber,
		"gasPrice":    gasPrice,
		"peerCount":   peerCount,
		"syncing":     syncing,
		"timestamp":   time.Now().Unix(),
	}

	data, _ := json.Marshal(stats)
	_, err := dt.js.Publish("eth.network", data)
	return err
}

// Publish current gas price
func (dt *SomniaStream) publishGasPrice() error {
	gasPrice, err := dt.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	gasPriceData := map[string]interface{}{
		"gasPrice":  gasPrice.String(),
		"gwei":      float64(gasPrice.Uint64()) / 1e9,
		"timestamp": time.Now().Unix(),
	}

	data, _ := json.Marshal(gasPriceData)
	_, err = dt.js.Publish("eth.gasPrice", data)
	return err
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Handle WebSocket for specific stream
// func (dt *DevTool) handleWebSocketStream(c *gin.Context) {
// 	stream := c.Param("stream")
// 	subject := dt.getStreamSubject(stream)

// 	conn, err := dt.upgrader.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		return
// 	}
// 	defer conn.Close()

// 	// Subscribe to specific JetStream
// 	sub, _ := dt.js.Subscribe(subject, func(msg *nats.Msg) {
// 		conn.WriteMessage(websocket.TextMessage, msg.Data)
// 		msg.Ack() // Acknowledge message
// 	}, nats.DeliverNew())
// 	defer sub.Unsubscribe()

// 	// Keep connection alive
// 	for {
// 		if _, _, err := conn.ReadMessage(); err != nil {
// 			break
// 		}
// 	}
// }

// Handle SSE for specific stream
func (dt *SomniaStream) handleSSEStream(c *gin.Context) {
	stream := c.Param("stream")
	subject := dt.getStreamSubject(stream)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Subscribe to specific JetStream
	sub, _ := dt.js.Subscribe(subject, func(msg *nats.Msg) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Data)
		c.Writer.Flush()
		msg.Ack() // Acknowledge message
	}, nats.DeliverNew())
	defer sub.Unsubscribe()

	// Keep connection alive
	<-c.Request.Context().Done()
}

// List available streams
func (dt *SomniaStream) listStreams(c *gin.Context) {
	streams := map[string]string{
		"blocks":        "eth.blocks.full - Full block data with transactions (JetStream)",
		"pending":       "eth.pending - Pending transactions (JetStream)",
		"logs":          "eth.logs - Recent event logs (JetStream)",
		"network":       "eth.network - Network statistics (JetStream)",
		"gasPrice":      "eth.gasPrice - Current gas price (JetStream)",
		"blocks-simple": "eth.blocks - Simple block data (JetStream)",
	}

	c.JSON(200, gin.H{
		"streams": streams,
		"usage": map[string]string{
			"websocket": "/ws/:stream (e.g., /ws/blocks)",
			"sse":       "/sse/:stream (e.g., /sse/pending)",
			"all_ws":    "/ws (subscribes to eth.blocks.full)",
			"all_sse":   "/sse (subscribes to eth.blocks.full)",
		},
		"jetstream": "All streams use NATS JetStream for persistence and replay",
	})
}

// Get NATS subject for stream name
func (dt *SomniaStream) getStreamSubject(stream string) string {
	switch stream {
	case "blocks":
		return "eth.blocks.full"
	case "pending":
		return "eth.pending"
	case "logs":
		return "eth.logs"
	case "network":
		return "eth.network"
	case "gasPrice", "gas":
		return "eth.gasPrice"
	case "blocks-simple":
		return "eth.blocks"
	default:
		return "eth.blocks.full" // Default fallback
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env file, using environment variables and defaults")
	} else {
		log.Println("✅ Loaded configuration from .env file")
	}

	// Initialize configuration
	config := &Config{
		RPCEndpoint: getEnv("RPC_ENDPOINT", "https://dream-rpc.somnia.network"),
		NATSUrl:     getEnv("NATS_URL", "nats://localhost:4222"),
		NATSToken:   getEnv("NATS_TOKEN", "nats_token"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
	}

	// Initialize the devtool
	devtool, err := NewSomniaStream(config)
	if err != nil {
		log.Fatalf("Failed to initialize devtool: %v", err)
	}

	// Start the devtool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	if err := devtool.Start(ctx); err != nil {
		log.Fatalf("Failed to start devtool: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
