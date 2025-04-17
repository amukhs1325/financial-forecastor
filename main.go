/*
Package main implements a service that periodically scrapes stock data from Yahoo Finance,
stores the historical values, and requests price predictions from a separate Python-based
machine learning microservice.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gorilla/mux"
)

/*
StockData represents a single snapshot of a stock's market data,
including the symbol, current price, volume, and timestamp.
*/
type StockData struct {
    Symbol    string    `json:"symbol"`
    Price     float64   `json:"price"`
    Volume    int64     `json:"volume"`
    Timestamp time.Time `json:"timestamp"`
}

/*
Prediction holds the output from the ML service, including the symbol,
current and predicted prices, and the percentage change.
*/
type Prediction struct {
    Symbol              string    `json:"symbol"`
    CurrentPrice        float64   `json:"current_price"`
    PredictedPrice      float64   `json:"predicted_price"`
    PredictedChange     float64   `json:"predicted_change"`
    PredictedChangePerc float64   `json:"predicted_change_percent"`
    Timestamp           time.Time `json:"timestamp"`
}

/*
DataCollector encapsulates a Colly collector to fetch stock data from Yahoo Finance.
*/
type DataCollector struct {
    collector *colly.Collector
}

/*
NewDataCollector initializes a Colly collector with a random delay and proper headers
to safely scrape Yahoo Finance data.
*/
func NewDataCollector() *DataCollector {
    c := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0"),
        colly.AllowedDomains("finance.yahoo.com"),
    )
    c.Limit(&colly.LimitRule{DomainGlob: "*", RandomDelay: 5 * time.Second})
    return &DataCollector{collector: c}
}

/*
CleanNumberString removes commas and trims whitespace, preparing a numeric string
for parsing into floats or integers.
*/
func CleanNumberString(s string) string {
    s = strings.ReplaceAll(s, ",", "")
    return strings.TrimSpace(s)
}

/*
FetchStockData visits the Yahoo Finance quote page for the given symbol,
extracts the regular market price and volume, and returns a StockData struct.
*/
func (dc *DataCollector) FetchStockData(symbol string) (*StockData, error) {
    sd := &StockData{Symbol: symbol, Timestamp: time.Now()}

    c := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0"),
        colly.AllowedDomains("finance.yahoo.com"),
    )

    url := fmt.Sprintf("https://finance.yahoo.com/quote/%s", symbol)
    c.OnHTML("fin-streamer[data-field='regularMarketPrice']", func(e *colly.HTMLElement) {
        txt := e.Text
        if txt == "" {
            txt = e.Attr("value")
        }
        if txt != "" {
            if v, err := strconv.ParseFloat(CleanNumberString(txt), 64); err == nil {
                sd.Price = v
            }
        }
    })
    c.OnHTML("fin-streamer[data-field='regularMarketVolume']", func(e *colly.HTMLElement) {
        txt := e.Text
        if txt == "" {
            txt = e.Attr("value")
        }
        if txt != "" {
            if v, err := strconv.ParseInt(CleanNumberString(txt), 10, 64); err == nil {
                sd.Volume = v
            }
        }
    })

    if err := c.Visit(url); err != nil {
        return nil, err
    }
    c.Wait()

    // Fallback or further parsing omitted for brevity
    return sd, nil
}

/*
FinancialProcessor manages concurrent data collection for multiple symbols
and forwards batches to the ML microservice for prediction.
*/
type FinancialProcessor struct {
    collectors map[string]*DataCollector
    dataStore  map[string][]StockData
    symbols    []string
    mutex      sync.RWMutex
    wg         sync.WaitGroup
}

/*
NewFinancialProcessor initializes the processor with a list of symbols to track.
*/
func NewFinancialProcessor(symbols []string) *FinancialProcessor {
    cols := make(map[string]*DataCollector)
    for _, s := range symbols {
        cols[s] = NewDataCollector()
    }
    return &FinancialProcessor{
        collectors: cols,
        dataStore:  make(map[string][]StockData),
        symbols:    symbols,
    }
}

/*
Start launches a goroutine for each symbol to periodically scrape and predict.
*/
func (fp *FinancialProcessor) Start() {
    for _, sym := range fp.symbols {
        fp.wg.Add(1)
        go fp.periodicCollection(sym)
    }
}

/*
periodicCollection fetches new data every 30s, stores up to 100 points,
and triggers prediction once enough history is collected.
*/
func (fp *FinancialProcessor) periodicCollection(symbol string) {
    defer fp.wg.Done()
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // Initial fetch
    if sd, err := fp.collectors[symbol].FetchStockData(symbol); err == nil {
        fp.mutex.Lock()
        fp.dataStore[symbol] = append(fp.dataStore[symbol], *sd)
        fp.mutex.Unlock()
        if len(fp.dataStore[symbol]) >= 5 {
            go fp.getPrediction(symbol)
        }
    }

    for range ticker.C {
        if sd, err := fp.collectors[symbol].FetchStockData(symbol); err == nil {
            fp.mutex.Lock()
            arr := fp.dataStore[symbol]
            arr = append(arr, *sd)
            if len(arr) > 100 {
                arr = arr[len(arr)-100:]
            }
            fp.dataStore[symbol] = arr
            fp.mutex.Unlock()
            go fp.getPrediction(symbol)
        }
    }
}

/*
getPrediction sends the last batch of data to the ML service
and logs the returned Prediction struct.
*/
func (fp *FinancialProcessor) getPrediction(symbol string) {
    fp.mutex.RLock()
    data := fp.dataStore[symbol]
    fp.mutex.RUnlock()
    if len(data) < 5 {
        return
    }

    payload := map[string]interface{}{"symbol": symbol, "data": data}
    body, _ := json.Marshal(payload)

    host := os.Getenv("ML_SERVICE_HOST")
    if host == "" {
        host = "localhost"
    }
    port := os.Getenv("ML_PORT")
    if port == "" {
        port = "5001"
    }
    url := fmt.Sprintf("http://%s:%s/predict", host, port)

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
    if err != nil {
        log.Printf("prediction error: %v", err)
        return
    }
    defer resp.Body.Close()

    var p Prediction
    if err := json.NewDecoder(resp.Body).Decode(&p); err == nil {
        log.Printf("Prediction for %s: %.2f → %.2f (%.2f%%)",
            p.Symbol, p.CurrentPrice, p.PredictedPrice, p.PredictedChangePerc)
    }
}

/*
handleGetData exposes an HTTP GET endpoint to retrieve stored history
for a given symbol.
*/
func (fp *FinancialProcessor) handleGetData(w http.ResponseWriter, r *http.Request) {
    sym := mux.Vars(r)["symbol"]
    fp.mutex.RLock()
    data, ok := fp.dataStore[sym]
    fp.mutex.RUnlock()
    if !ok {
        http.Error(w, "no data", http.StatusNotFound)
        return
    }
    json.NewEncoder(w).Encode(data)
}

/*
main initializes the FinancialProcessor, starts scraping/prediction routines,
and runs the HTTP server on the configured port.
*/
func main() {
    symbols := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "META"}
    fp := NewFinancialProcessor(symbols)
    fp.Start()

    r := mux.NewRouter()
    r.HandleFunc("/api/data/{symbol}", fp.handleGetData).Methods("GET")

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("Listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
