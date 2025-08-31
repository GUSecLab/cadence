FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3.12-venv \
    golang-go \
    build-essential \
    git \
    sudo \
    ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Set Go environment variables
ENV PATH="/usr/lib/go/bin:${PATH}"

# Create Python virtual environment and add to PATH
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:${PATH}"

# Set working directory
WORKDIR /app
COPY . .

WORKDIR /app/scripts
RUN pip install --upgrade pip && pip install -r requirements.txt

WORKDIR /app
RUN python3 scripts/run_setup.py
RUN python3 scripts/run_build.py
RUN python3 scripts/run_import.py

VOLUME /app/results

# Default command
CMD ["sh", "-c", "mkdir -p /app/results/raw-data /app/results/plots && python3 scripts/run_clear_dbs.py && python3 scripts/run_clear_results.py && python3 scripts/run_experiments_parrallel.py && python3 scripts/run_plots.py" ]
