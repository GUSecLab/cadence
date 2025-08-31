import os
import json
import math
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.backends.backend_pdf import PdfPages
import logging

from parameters import message_type

# Set up logging
logging.basicConfig(
    level=logging.INFO,
    format='[%(levelname)s] %(message)s'
)

SECONDS_PER_DAY = 86400

def generate_day_bins(start_time, end_time, days_per_bin=1):
    seconds_per_bin = days_per_bin * SECONDS_PER_DAY
    total_seconds = end_time - start_time
    num_bins = math.ceil(total_seconds / seconds_per_bin)
    bin_edges = [start_time + i * seconds_per_bin for i in range(num_bins + 1)]
    labels = [str(i + 1) for i in range(num_bins)]
    return bin_edges, labels, num_bins

def plot_originated_vs_delivered_messages(df_delivered, json_path, pdf_path, bin_edges, labels, days_per_bin, experiment_name):
    logging.info(f"Plotting originated vs delivered for {experiment_name}")
    
    # Delivered messages
    exp_data = df_delivered[df_delivered["experiment_name"] == experiment_name]
    delivered_times = exp_data["deliver_time"]
    num_buckets = len(labels)
    delivered_counts, _ = np.histogram(delivered_times, bins=bin_edges)

    # Originated messages
    with open(json_path, "r") as f:
        json_data = json.load(f)
    all_messages = []
    for message_list in json_data.values():
        all_messages.extend(message_list)
    df_originated = pd.DataFrame(all_messages)
    originated_times = df_originated["time"]
    originated_counts, _ = np.histogram(originated_times, bins=bin_edges)

    # Plot
    plt.figure(figsize=(10, 5))
    plt.bar(range(num_buckets), delivered_counts, width=0.8, tick_label=labels, label="Messages Delivered")
    plt.plot(
        range(num_buckets), originated_counts,
        label="Messages Originated", marker='o', linestyle='-', linewidth=2, color='red'
    )
    plt.xticks(rotation=45, ha="right")
    plt.xlabel(f"{days_per_bin}-Day Window")
    plt.ylabel("Number of Messages")
    plt.legend()
    plt.tight_layout()
    plt.savefig(pdf_path)
    plt.close()
    logging.info(f"  Plot saved to {pdf_path}")

def main():
    logging.info("Starting originated vs delivered analysis...")

    # Experiment time ranges
    japan_experiment_start = 1704067200.0
    japan_experiment_end = 1710545400.0
    tdrive_experiment_start = 1201959058.0
    tdrive_experiment_end = 1202492359.0

    output_dir = "results/plots/"
    os.makedirs(output_dir, exist_ok=True)

    # Load CSVs
    logging.info("Loading CSV files...")
    df_japan_delivered = pd.read_csv("results/raw-data/japan_delivered_messages.csv")
    df_tdrive_delivered = pd.read_csv("results/raw-data/tdrive_delivered_messages.csv")

    # Load JSONs
    japan_json_path = f"message_files/messages_japan_type_{message_type}.json"
    tdrive_json_path = f"message_files/messages_tdrive_type_{message_type}.json"

    # JAPAN
    if os.path.exists(japan_json_path):
        japan_days_per_bin = 5
        japan_bin_edges, japan_labels, _ = generate_day_bins(japan_experiment_start, japan_experiment_end, days_per_bin=japan_days_per_bin)
        japan_experiments = df_japan_delivered["experiment_name"].unique()
        for experiment_name in japan_experiments:
            filename = f"originated_vs_delivered_{experiment_name.replace('=', '_').replace(',', '_')}_japan.pdf"
            plot_originated_vs_delivered_messages(
                df_japan_delivered,
                japan_json_path,
                os.path.join(output_dir, filename),
                japan_bin_edges,
                japan_labels,
                japan_days_per_bin,
                experiment_name
            )

    # TDRIVE
    if os.path.exists(tdrive_json_path):
        tdrive_days_per_bin = 1
        tdrive_bin_edges, tdrive_labels, _ = generate_day_bins(tdrive_experiment_start, tdrive_experiment_end, days_per_bin=tdrive_days_per_bin)
        tdrive_experiments = df_tdrive_delivered["experiment_name"].unique()
        for experiment_name in tdrive_experiments:
            filename = f"originated_vs_delivered_{experiment_name.replace('=', '_').replace(',', '_')}_tdrive.pdf"
            plot_originated_vs_delivered_messages(
                df_tdrive_delivered,
                tdrive_json_path,
                os.path.join(output_dir, filename),
                tdrive_bin_edges,
                tdrive_labels,
                tdrive_days_per_bin,
                experiment_name
            )

    logging.info("All originated vs delivered plots generated.")

if __name__ == "__main__":
    main()
