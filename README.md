# financial-forecastor
Financial Forecastor

Overview: Financial Forecastor is a containerized microservice suite designed for real‑time stock data scraping and machine‑learning‑driven price predictions. The core scraper and API are implemented in Go, while model training and inference are handled by a Python/Flask service. Both services are orchestrated together via Docker Compose.

Features: The Go service fetches price and volume data from Yahoo Finance every thirty seconds and maintains the last one hundred data points per symbol in memory. The Python service trains a Random Forest regression model using features such as simple moving average and one‑day percent change. Models are retrained in the background as new data accumulates. Both services expose HTTP endpoints for retrieving history, submitting data for training or prediction, and obtaining results.

Prerequisites: To run this project you need Docker and Docker Compose, or local installations of Go version 1.24 or higher and Python version 3.9 or higher with pip for dependency management.

Configuration: Environment variables control ports and service discovery. The PORT variable sets the HTTP port for the Go backend. The ML_SERVICE_HOST and ML_PORT variables specify how the Go service connects to the Python/Flask service.

Installation and Quick Start: Clone the repository to your machine. To deploy with containers, navigate into the docker directory and use Docker Compose to build and start both services together. For local development without Docker, install Python dependencies via pip, launch the ML service on the configured port, then resolve Go modules and run the Go service pointing it at the ML service host and port.

API Endpoints: The Go backend provides an HTTP GET endpoint at /api/data/{symbol} which returns cached historical data for the given stock symbol. The Python service offers an HTTP POST endpoint at /predict to accept symbol data for training or prediction, and an HTTP GET endpoint at /data/{symbol} to retrieve raw stored data.

Architecture Overview: The Go service continuously scrapes market data and stores it in memory. When at least five data points are available, it forwards a batch to the Python service over HTTP. The Python service trains or predicts using its regression model and returns results to the Go service, which logs predictions and continues scraping.

Development and Testing: Maintain code quality with Go and Python linters. Implement unit tests for the scraping logic, prediction routines, and HTTP handlers, as well as integration tests that exercise both services together.

Project Structure: The repository includes a docker folder containing the Docker Compose file and Dockerfiles for each service, a main.go source file with the Go backend, go.mod and go.sum for Go dependencies, ml_service.py and requirements.txt for the Python service, an optional predictor.proto schema file, and this README.




