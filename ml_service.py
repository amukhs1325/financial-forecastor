
"""
ml_service.py

This module implements a Flask-based microservice for training and predicting stock prices.
It exposes two endpoints:
  1. POST /predict   - Train or predict using incoming stock data
  2. GET  /data/<symbol> - Retrieve stored historical data for a given symbol

The service maintains in-memory stores for models and raw data. A background thread
periodically retrains models on accumulated data.
"""


from flask import Flask, request, jsonify
from flask_cors import CORS


import pandas as pd
from sklearn.ensemble import RandomForestRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.exceptions import NotFittedError


import os
from datetime import datetime
import time
import threading


app = Flask(__name__)
CORS(app)


models = {}
data_store = {}

class StockPriceModel:
    """
    Encapsulates a RandomForestRegressor for a given stock symbol,
    along with a StandardScaler for feature normalization.
    """

    def __init__(self, symbol):
        """
        Initialize model and scaler for the given symbol.
        """
        self.symbol = symbol
        self.model = RandomForestRegressor(n_estimators=100, random_state=42)
        self.scaler = StandardScaler()

    def _prepare_features(self, df):
        """
        Given a DataFrame with 'price' and 'volume' columns,
        compute technical features and the training target.

        Features:
          - price
          - volume
          - SMA_5 (5-period simple moving average)
          - price_change_1d (1-day percent change)

        Target:
          - next day's price

        Returns:
          X (DataFrame of features), y (Series of targets)
        """
        df['SMA_5'] = df['price'].rolling(window=5).mean()
        df['price_change_1d'] = df['price'].pct_change(1)
        df['target'] = df['price'].shift(-1)
        df = df.dropna()
        features = ['price', 'volume', 'SMA_5', 'price_change_1d']
        return df[features], df['target']

    def train(self, historical_data):
        """
        Train the model on historical_data, a list of dicts with keys:
          'symbol', 'price', 'volume', 'timestamp'.

        Returns a dict with training status and metadata or an error if insufficient data.
        """
        df = pd.DataFrame(historical_data)
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        df = df.sort_values('timestamp')

        X, y = self._prepare_features(df)
        if len(X) < 10:
            return {"error": "Not enough data for training"}

        X_scaled = self.scaler.fit_transform(X)
        self.model.fit(X_scaled, y)

        last_price = df['price'].iloc[-1]
        pred = self.predict(historical_data[-10:])
        return {
            "status": "success",
            "current_price": last_price,
            "predicted_price": pred["predicted_price"],
            "data_points": len(X)
        }

    def predict(self, current_data):
        """
        Predict the next price using the most recent slice of current_data.
        Returns a dict with symbol, current_price, predicted_price,
        absolute and percent change, and timestamp.
        """
        df = pd.DataFrame(current_data)
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        df = df.sort_values('timestamp')

        X, _ = self._prepare_features(df)
        if X.empty:
            return {"error": "Not enough data for prediction"}

        try:
            X_scaled = self.scaler.transform(X.iloc[-1:])
        except NotFittedError:
            
            return {"error": "Model not yet trained"}

        prediction = self.model.predict(X_scaled)[0]
        current_price = df['price'].iloc[-1]
        return {
            "symbol": self.symbol,
            "current_price": current_price,
            "predicted_price": prediction,
            "predicted_change": prediction - current_price,
            "predicted_change_percent": (prediction - current_price) / current_price * 100,
            "timestamp": datetime.now().isoformat()
        }

def background_training():
    """
    Background thread that periodically retrains all models
    once enough data (>=20 points) is collected per symbol.
    """
    while True:
        for symbol, data in data_store.items():
            if len(data) >= 20:
                if symbol not in models:
                    models[symbol] = StockPriceModel(symbol)
                models[symbol].train(data)
                print(f"Trained model for {symbol} with {len(data)} data points")
        time.sleep(30)


threading.Thread(target=background_training, daemon=True).start()

@app.route('/predict', methods=['POST'])
def predict_endpoint():
    """
    POST /predict
    Body JSON: { "symbol": <symbol>, "data": [ {symbol, price, volume, timestamp}, ... ] }

    - Stores incoming data in data_store.
    - If no model exists, attempts initial training.
    - Returns prediction or pending status if still training.
    """
    payload = request.json or {}
    symbol = payload.get('symbol')
    stock_data = payload.get('data')
    if not symbol or not stock_data:
        return jsonify({"error": "Symbol and data required"}), 400


    data_store[symbol] = stock_data

   
    if symbol not in models:
        candidate = StockPriceModel(symbol)
        result = candidate.train(stock_data)
        if "error" in result:
            return jsonify({"error": result["error"], "status": "pending_training"}), 200
        models[symbol] = candidate

   
    prediction = models[symbol].predict(stock_data)
    if "error" in prediction:
        return jsonify(prediction), 200
    return jsonify(prediction)

@app.route('/data/<symbol>', methods=['GET'])
def get_data(symbol):
    """
    GET /data/<symbol>
    Returns the raw historical data for the given symbol, or an error if none exists.
    """
    if symbol not in data_store:
        return jsonify({"error": "No data for this symbol"}), 404
    return jsonify(data_store[symbol])

if __name__ == '__main__':
    """
    Entry point for the Flask app.
    Uses the ML_PORT environment variable (default 5001).
    """
    port = int(os.environ.get('ML_PORT', 5001))
    app.run(host='0.0.0.0', port=port, debug=True)
