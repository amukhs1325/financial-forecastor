# financial-forecastor
📈 Financial Forecastor
A containerized microservice suite for real‑time stock data scraping and machine‑learning‑driven price predictions.
Built with Go (scraper & API) and Python/Flask (model training & inference), orchestrated via Docker Compose.

🚀 Features
Real‑time Scraping
Fetches price & volume for configurable symbols from Yahoo Finance every 30 seconds with resilient parsing and fallback mocks.

History Cache
Stores the last 100 data points per symbol in memory and exposes them via REST.

Machine Learning Service
Random Forest regression with standardized features (SMA, percent change), background retraining thread, and HTTP endpoints for training status, predictions, and raw data.

Containerized Deployment
Two Dockerfiles (Dockerfile for Go, Dockerfile.python for Python) and a single docker-compose.yml to build/run both services on a bridge network.

🔧 Prerequisites
Docker & Docker Compose

Go 1.24+ (if running Go service locally)

Python 3.9+ & pip (if running ML service locally)

⚙️ Configuration

Environment Variable	Description	Default
PORT	Port for Go backend HTTP server	8080
ML_SERVICE_HOST	Hostname/IP for ML service	ml-service/localhost
ML_PORT	Port for ML service HTTP server	5001
📦 Installation & Quick Start
Clone the repository
bash
Copy
Edit
git clone https://github.com/yourusername/financial-forecastor.git
cd financial-forecastor
Using Docker Compose (recommended)
bash
Copy
Edit
cd docker
docker-compose up --build
Go backend → http://localhost:8081

ML service → http://localhost:5001

Run in background:

bash
Copy
Edit
docker-compose up -d
Local Development (without Docker)
Start the Python ML service
bash
Copy
Edit
pip install -r requirements.txt
export ML_PORT=5001
python3 ml_service.py
Start the Go backend
bash
Copy
Edit
go mod download
export PORT=8080
export ML_SERVICE_HOST=localhost
export ML_PORT=5001
go run main.go
🔍 API Endpoints
Go Backend (port 8081)
GET /api/data/{symbol}
Returns cached history for the given stock symbol.

bash
Copy
Edit
curl http://localhost:8081/api/data/AAPL
ML Service (port 5001)
POST /predict
Train or predict on incoming data.

bash
Copy
Edit
curl -X POST http://localhost:5001/predict \
     -H "Content-Type: application/json" \
     -d '{"symbol":"AAPL","data":[{"symbol":"AAPL","price":150,"volume":1200000,"timestamp":"2025-04-17T12:00:00Z"}]}'
GET /data/{symbol}
Retrieve raw historical data.

bash
Copy
Edit
curl http://localhost:5001/data/AAPL
🏗 Architecture Overview
typescript
Copy
Edit
Go Backend (8080)
  ├─ Scrape Yahoo Finance every 30s
  ├─ Cache last 100 points per symbol
  └─ GET /api/data/{symbol}

      │ POST batch ≥5 points
      ▼

ML Service (5001)
  ├─ POST /predict
  ├─ Train & predict with RandomForest
  └─ GET /data/{symbol}
🛠 Development & Testing
Go linting: golangci-lint run

Python linting: flake8 ml_service.py

Unit tests: (add your test commands here)

📁 Project Structure
python
Copy
Edit
financial-forecastor/
├── docker/
│   ├── docker-compose.yml
│   ├── Dockerfile            # Go service
│   └── Dockerfile.python     # Python ML service
├── main.go                   # Go backend source
├── go.mod / go.sum           # Go dependencies
├── ml_service.py             # Python ML service
├── requirements.txt          # Python dependencies
├── predictor.proto           # (optional) gRPC schema
└── README.md                 # Project documentation
📜 License
This project is licensed under the MIT License.
