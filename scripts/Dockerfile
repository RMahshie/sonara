FROM python:3.11-slim
WORKDIR /app

# Install uv for faster package installation
RUN pip install --no-cache-dir uv

COPY requirements.txt .
RUN uv pip install --system -r requirements.txt
COPY . .