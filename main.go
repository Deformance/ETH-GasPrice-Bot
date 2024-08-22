package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const endpoint = "https://api.etherscan.io/api?module=gastracker&action=gasoracle&apikey="
const shards = 1

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		LastBlock       string `json:"LastBlock"`
		SafeGasPrice    string `json:"SafeGasPrice"`
		ProposeGasPrice string `json:"ProposeGasPrice"`
		FastGasPrice    string `json:"FastGasPrice"`
		SuggestBaseFee  string `json:"suggestBaseFee"`
		GasUsedRatio    string `json:"gasUsedRatio"`
	} `json:"result"`
}

func getEnvOrDie(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading env: %v", err)
	}

	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Could not find %v in .env", key)
	}

	return value
}

func getPrices(api_key string) (string, error) {
	res, err := http.Get(endpoint + api_key)

	if res != nil {
		defer res.Body.Close()
	}

	if err != nil {
		return "", fmt.Errorf("failed to fetch: %v", err)
	}

	jsonPayload, err := decodeJson[Response](res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode json: %v", err)
	}

	slow_amount, err := strconv.ParseFloat(jsonPayload.Result.SafeGasPrice, 64)
	if err != nil {
		return "", fmt.Errorf("invalid slow amount format: %v", err)
	}

	mid_amount, err := strconv.ParseFloat(jsonPayload.Result.ProposeGasPrice, 64)
	if err != nil {
		return "", fmt.Errorf("invalid mid amount format: %v", err)
	}

	fast_amount, err := strconv.ParseFloat(jsonPayload.Result.FastGasPrice, 64)
	if err != nil {
		return "", fmt.Errorf("invalid fast amount format: %v", err)
	}

	slow_price := fmt.Sprintf("%.0f", slow_amount)
	mid_price := fmt.Sprintf("%.0f", mid_amount)
	fast_price := fmt.Sprintf("%.0f", fast_amount)
	return "🚀 " + fast_price + " | 🐦 " + mid_price + " | 🐌 " + slow_price, err
}

func decodeJson[T any](r io.Reader) (T, error) {
	var v T
	err := json.NewDecoder(r).Decode(&v)
	return v, err
}

func worker(id int, token string, api_key string) {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	discord.ShardCount = shards
	discord.ShardID = id

	err = discord.Open()
	if err != nil {
		log.Fatalf("Error opening Discord ws: %v", err)
	}
	defer discord.Close()

	for {
		res, err := getPrices(api_key)
		if err != nil {
			log.Printf("Error getting gas price for shard %d: %v \n", id, err)
		} else {
			fmt.Printf("WorkerId %v got %v \n", id, res)
			err = discord.UpdateWatchStatus(0, res)
			if err != nil {
				log.Printf("Error updating Discord status for shard %d: %v \n", id, err)
			}
		}
		time.Sleep(30 * time.Second)
	}

}

func main() {
	fmt.Println("Hello world! 👋🌍")
	token := getEnvOrDie("TOKEN")
	api_key := getEnvOrDie("API_KEY")

	wg := sync.WaitGroup{}

	for shardId := 0; shardId < shards; shardId++ {
		wg.Add(1)
		go worker(shardId, token, api_key)
	}

	wg.Wait()
}
