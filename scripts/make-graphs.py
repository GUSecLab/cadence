import argparse
import numpy as np
import matplotlib
import csv
import matplotlib.pyplot as plt
import re
import duckdb
import pandas as pd
import math
import os

from parameters import p_values, k_values


def set_plot_options():
    options = {
        'font.size': 12,
        'figure.figsize': (5, 4),
        'figure.dpi': 100.0,
        'figure.subplot.left': 0.20,
        'figure.subplot.right': 0.97,
        'figure.subplot.bottom': 0.20,
        'figure.subplot.top': 0.90,
        'grid.color': '0.1',
        'grid.linestyle': ':',
        'axes.grid': True,
        'axes.titlesize': 'medium',
        'axes.labelsize': 'medium',
        'axes.formatter.limits': (-4, 4),
        'xtick.labelsize': 12,
        'ytick.labelsize': 12,
        'lines.linewidth': 2.0,
        'lines.markeredgewidth': 0.5,
        'lines.markersize': 10,
        'legend.fontsize': 11,
        'legend.fancybox': False,
        'legend.shadow': False,
        'legend.borderaxespad': 0.5,
        'legend.numpoints': 1,
        'legend.handletextpad': 0.5,
        'legend.handlelength': 2.0,
        'legend.labelspacing': .75,
        'legend.markerscale': 1.0,
        'ps.useafm': True,
        'pdf.use14corefonts': True,
        'text.usetex': True,
    }

    for option_key in options:
        matplotlib.rcParams[option_key] = options[option_key]

    if 'figure.max_num_figures' in matplotlib.rcParams:
        matplotlib.rcParams['figure.max_num_figures'] = 50
    if 'figure.max_open_warning' in matplotlib.rcParams:
        matplotlib.rcParams['figure.max_open_warning'] = 50
    if 'legend.ncol' in matplotlib.rcParams:
        matplotlib.rcParams['legend.ncol'] = 50


def parse_arguments():
    parser = argparse.ArgumentParser(description="plotter")
    parser.add_argument('--data-file', type=str, required=True)
    parser.add_argument('--output-file', type=str, required=True)
    parser.add_argument('--xlabel', type=str, default='\\boldmath$\\varepsilon$')
    parser.add_argument('--yrange', help="y range (in min:max format)")
    parser.add_argument('--logy', action='store_true')
    parser.add_argument('--metrics', nargs='+')
    return parser.parse_args()


def read_csv_to_dict(csv_file):
    values = []
    with open(csv_file, mode='r') as file:
        reader = csv.DictReader(file)
        for row in reader:
            exp_name = row['experiment_name']
            key_value_pairs = re.findall(r'(\w+)=([^_\s]+)', exp_name)
            key_value_dict = {key: float(value) for key, value in key_value_pairs}
            exp_name = re.sub(r'\w+=\S+', '', exp_name).strip()
            row_data = {'exp_name': exp_name}
            for k in key_value_dict:
                row_data[k] = key_value_dict[k]
            for col in reader.fieldnames[1:]:
                row_data[col] = float(row[col])
            values.append(row_data)
    return values


def plot_mirage_efficiency(data_df, args, metric_field, y_label, filename_prefix, filename_suffix):
    for k in k_values:
        fig = plt.figure(figsize=(5, 4))  # height = 4
        plt.clf()
        plt.xlabel(args.xlabel)
        plt.ylabel(y_label)

        x_values, medians, q1s, q3s = [], [], [], []

        for p in p_values:
            query = f"SELECT {metric_field} AS w FROM data_df WHERE k={k} AND p={p} AND exp_name LIKE '%mirage%'"
            df = duckdb.sql(query).df()
            if df.empty:
                continue
            epsilon = (k * (k - 1.0)) * math.log(p / (1.0 - p))
            x_values.append(epsilon)
            medians.append(df['w'].median())
            q1s.append(np.percentile(df['w'], 25))
            q3s.append(np.percentile(df['w'], 75))
            plt.annotate(f'p={p}', (epsilon, df['w'].median()), size='x-small', textcoords="offset points", xytext=(5, 15), ha='center')

        plt.plot(x_values, medians, marker='*')
        plt.fill_between(x_values, q1s, q3s, alpha=0.3)

        if args.yrange:
            ymin, ymax = map(float, args.yrange.split(':'))
            plt.ylim(ymin, ymax)
        if args.logy:
            plt.yscale('log')

        x_min, x_max = plt.xlim()
        x_buffer = 0.05 * (x_max - x_min)
        plt.xlim(x_min - x_buffer, x_max + x_buffer)

        plt.tight_layout(rect=[0, 0.15, 1, 1])
        out_dir = os.path.dirname(args.output_file)
        os.makedirs(out_dir, exist_ok=True)
        out_file = os.path.join(out_dir, f"{filename_prefix}-k_{k}{filename_suffix}-{os.path.basename(args.output_file)}")
        plt.savefig(out_file, bbox_inches='tight')
        print(f"Mirage-only plot saved to {out_file}")

def plot_combined_mirage_efficiency(data_df, args, metric_field, y_label, filename_prefix, filename_suffix):
    fig = plt.figure(figsize=(5, 4))
    plt.clf()
    plt.xlabel(args.xlabel)
    plt.ylabel(y_label)

    for k in k_values:
        x_values, medians, q1s, q3s = [], [], [], []

        for p in p_values:
            query = f"SELECT {metric_field} AS w FROM data_df WHERE k={k} AND p={p} AND exp_name LIKE '%mirage%'"
            df = duckdb.sql(query).df()
            if df.empty:
                continue
            epsilon = (k * (k - 1.0)) * math.log(p / (1.0 - p))
            x_values.append(epsilon)
            medians.append(df['w'].median())
            q1s.append(np.percentile(df['w'], 25))
            q3s.append(np.percentile(df['w'], 75))

        plt.plot(x_values, medians, marker='*', label=f"k={k}")
        plt.fill_between(x_values, q1s, q3s, alpha=0.2)

    if args.yrange:
        ymin, ymax = map(float, args.yrange.split(':'))
        plt.ylim(ymin, ymax)
    if args.logy:
        plt.yscale('log')

    x_min, x_max = plt.xlim()
    x_buffer = 0.05 * (x_max - x_min)
    plt.xlim(x_min - x_buffer, x_max + x_buffer)

    plt.legend()
    plt.tight_layout(rect=[0, 0.15, 1, 1])

    out_dir = os.path.dirname(args.output_file)
    os.makedirs(out_dir, exist_ok=True)
    out_file = os.path.join(out_dir, f"{filename_prefix}{filename_suffix}-combined-{os.path.basename(args.output_file)}")
    plt.savefig(out_file, bbox_inches='tight')
    print(f"Combined Mirage plot saved to {out_file}")


def main():
    args = parse_arguments()
    set_plot_options()
    data = read_csv_to_dict(args.data_file)
    data_df = pd.DataFrame(data)

    all_metrics = {
        'dr': {'name': 'Delivery Rate', 'field': 'throughput', 'short': 'dr'},
        'nl': {'name': 'Network Load', 'field': 'net_load', 'short': 'nl'},
        'al': {'name': 'Message Load', 'field': 'average_load', 'short': 'al'},
        'custom': {'name': 'Message Load Efficiency', 'field': 'throughput/average_load', 'short': 'custom'},
        'custom2': {'name': 'Delivery Efficiency', 'field': 'throughput/net_load', 'short': 'custom2'},
        'lat': {'name': 'Latency (days)', 'field': 'lat_sec/(24.0*60.0*60.0)', 'short': 'lat'},
    }

    selected_metrics = args.metrics or all_metrics.keys()

    for metric_name_short in selected_metrics:
        metric = all_metrics[metric_name_short]
        is_custom = metric_name_short in ['custom', 'custom2']
        configs = [{'exclude': False}] + ([{'exclude': True}] if is_custom else [])

        for config in configs:
            exclude = config['exclude']
            title_extra = ""
            filename_extra = "-exclude" if exclude else ""

            handoff_value = duckdb.sql(f"SELECT {metric['field']} AS w FROM data_df WHERE exp_name LIKE '%randomwalk-v1'").fetchone()[0]
            flooding_value = duckdb.sql(f"SELECT {metric['field']} AS w FROM data_df WHERE exp_name LIKE '%broadcast'").fetchone()[0]
            randomwalk = duckdb.sql(f"SELECT {metric['field']} AS w FROM data_df WHERE exp_name LIKE '%randomwalk-v1-random'").df()['w']
            ppbr = duckdb.sql(f"SELECT {metric['field']} AS w FROM data_df WHERE exp_name LIKE '%ppbr'").df()['w']
            med_rw, q1_rw, q3_rw = np.median(randomwalk), np.percentile(randomwalk, 25), np.percentile(randomwalk, 75)
            med_ppbr, q1_ppbr, q3_ppbr = np.median(ppbr), np.percentile(ppbr, 25), np.percentile(ppbr, 75)

            if is_custom:
                flood_dr = duckdb.sql("SELECT throughput FROM data_df WHERE exp_name LIKE '%broadcast'").fetchone()[0]
                ppbr_dr = duckdb.sql("SELECT throughput FROM data_df WHERE exp_name LIKE '%ppbr'").df()['throughput'].median()
                rw_dr = duckdb.sql("SELECT throughput FROM data_df WHERE exp_name LIKE '%randomwalk-v1-random'").df()['throughput'].median()
                ho_dr = duckdb.sql("SELECT throughput FROM data_df WHERE exp_name LIKE '%randomwalk-v1'").fetchone()[0]

            for k in k_values:
                fig = plt.figure(figsize=(5, 5.5)) if is_custom else plt.figure()
                plt.clf()
                plt.xlabel(args.xlabel)
                plt.ylabel(metric['name'] + title_extra)
                x_values, medians, q1s, q3s, mirage_drs = [], [], [], [], []

                for p in p_values:
                    query = f"SELECT {metric['field']} AS w, throughput FROM data_df WHERE k={k} AND p={p} AND exp_name LIKE '%mirage%'"
                    df = duckdb.sql(query).df()
                    if df.empty: continue
                    epsilon = (k * (k - 1.0)) * math.log(p / (1.0 - p))
                    x_values.append(epsilon)
                    medians.append(df['w'].median())
                    q1s.append(np.percentile(df['w'], 25))
                    q3s.append(np.percentile(df['w'], 75))
                    if is_custom:
                        mirage_drs.append(df['throughput'].median())
                    plt.annotate(f'p={p}', (epsilon, df['w'].median()), size='x-small', textcoords="offset points", xytext=(5, 15), ha='center')

                plt.plot(x_values, [flooding_value]*len(x_values), marker='.', label=f"Max.~Flooding (DR={flood_dr:.2f})" if is_custom else "Max.~Flooding")
                plt.plot(x_values, medians, marker='*', label=f"Mirage (median, DR={np.median(mirage_drs):.2f})" if is_custom else "Mirage (median)")
                plt.fill_between(x_values, q1s, q3s, alpha=0.3)

                if not exclude:
                    plt.plot(x_values, [med_rw]*len(x_values), marker='^', label=f"Prob.~Flooding (DR={rw_dr:.2f})" if is_custom else "Prob.~Flooding")
                    plt.fill_between(x_values, [q1_rw]*len(x_values), [q3_rw]*len(x_values), alpha=0.3)
                plt.plot(x_values, [med_ppbr]*len(x_values), marker='|', label=f"PPBR (DR={ppbr_dr:.2f})" if is_custom else "PPBR")
                plt.fill_between(x_values, [q1_ppbr]*len(x_values), [q3_ppbr]*len(x_values), alpha=0.3)

                if not exclude and metric_name_short != "lat":
                    plt.plot(x_values, [handoff_value]*len(x_values), marker='x', label=f"Handoff (DR={ho_dr:.2f})" if is_custom else "Handoff")

                if args.yrange:
                    ymin, ymax = map(float, args.yrange.split(':'))
                    plt.ylim(ymin, ymax)
                if args.logy:
                    plt.yscale('log')

                x_min, x_max = plt.xlim()
                x_buffer = 0.05 * (x_max - x_min)
                plt.xlim(x_min - x_buffer, x_max + x_buffer)

                plt.legend(loc='upper center', bbox_to_anchor=(0.5, -0.25), ncol=2) if is_custom else plt.legend()
                plt.tight_layout(rect=[0, 0.15, 1, 1]) if is_custom else plt.tight_layout()

                out_dir = os.path.dirname(args.output_file)
                os.makedirs(out_dir, exist_ok=True)
                out_file = os.path.join(out_dir, f"{metric['short']}-k_{k}{filename_extra}-{os.path.basename(args.output_file)}")
                plt.savefig(out_file, bbox_inches='tight')
                print(f"plot saved to {out_file}")

    # Additional Mirage-only efficiency plots
    plot_mirage_efficiency(data_df, args, "throughput/net_load", "Delivery Efficiency", "delivery-efficiency", "-transfers")
    plot_mirage_efficiency(data_df, args, "throughput/net_load", "Delivery Efficiency", "delivery-efficiency", "-transfers")
    plot_mirage_efficiency(data_df, args, "throughput/average_load", "Message Load Efficiency", "message-efficiency", "-copies")
    plot_mirage_efficiency(data_df, args, "throughput/average_load", "Message Load Efficiency", "message-efficiency", "-copies")

    # Combined overlay plots with all k lines
    plot_combined_mirage_efficiency(data_df, args, "throughput/net_load", "Delivery Efficiency", "delivery-efficiency", "-transfers")
    plot_combined_mirage_efficiency(data_df, args, "throughput/average_load", "Message Load Efficiency", "message-efficiency", "-copies")


if __name__ == "__main__":
    main()
