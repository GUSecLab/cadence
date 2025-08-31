import json
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor

from parameters import p_values, k_values, buffer_size_values, message_type  # Import p and k values


def run_command(command):
    """Run a shell command and exit if it fails."""
    print(f"Executing: {command}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Error: Command failed - {command}")
        # sys.exit(1)


def modify_json(file_path, updates):
    """
    Modify a JSON file with the given updates.

    :param file_path: Path to the JSON file.
    :param updates: Dictionary containing the updates to apply.
    """
    try:
        # Load the JSON file
        with open(file_path, "r") as f:
            data = json.load(f)

        # Apply updates
        for key, value in updates.items():
            if isinstance(value, dict) and key in data:
                data[key].update(value)
            else:
                data[key] = value

        # Save the modified JSON file
        with open(file_path, "w") as f:
            json.dump(data, f, indent=4)
        print(f"Successfully updated {file_path}")

    except Exception as e:
        print(f"Error modifying JSON file: {e}")
        sys.exit(1)


def run_mirage():
    def run_for_dataset(config_file, logic_config_file, dataset_name):
        run_values = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
        for buffer_size in buffer_size_values:
            for p in p_values:
                for k in k_values:
                    # Update the logic config file with the current p and k values
                    modify_json(logic_config_file, {
                        "p": p,
                        "k": k
                    })

                    for run in run_values:
                        # Generate a unique experiment name based on p, k, and run
                        experiment_name = f"{dataset_name} - mirage p={p} k={k} buffer={buffer_size} run={run}"
                        messages_file_name = f"message_files/messages_{dataset_name}_type_{message_type}.json"

                        # Update the JSON file with the current run and experiment name
                        modify_json(config_file, {
                            "simulation": {
                                "experiment_name": experiment_name,
                                "min_buffer_size": buffer_size,
                                "max_buffer_size": buffer_size,
                                "message_generation_type": message_type,
                                "messages_file": messages_file_name
                            }
                        })

                        # Run the simulation command
                        command = f"./cmd/cadence/cadence sim {config_file}"
                        run_command(command)

    datasets = [
        ("cmd/cadence/configs/NDSS/config_japan_mirage.json", "pkg/logic/logic_configs/NDSS/logic_config_japan_mirage.json", "japan"),
        ("cmd/cadence/configs/NDSS/config_tdrive_mirage.json", "pkg/logic/logic_configs/NDSS/logic_config_tdrive_mirage.json", "tdrive")
    ]

    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_for_dataset, *dataset) for dataset in datasets]
        for future in futures:
            future.result()


def run_ppbr():
    def run_for_dataset(config_file, dataset_name):
        run_values = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
        for buffer_size in buffer_size_values:
            for run in run_values:
                experiment_name = f"{dataset_name} - ppbr buffer={buffer_size} run={run}"
                messages_file_name = f"message_files/messages_{dataset_name}_type_{message_type}.json"

                modify_json(config_file, {
                    "simulation": {
                        "experiment_name": experiment_name,
                        "min_buffer_size": buffer_size,
                        "max_buffer_size": buffer_size,
                        "message_generation_type": message_type,
                        "messages_file": messages_file_name
                    }
                })
                command = f"./cmd/cadence/cadence sim {config_file}"
                run_command(command)

    datasets = [
        ("cmd/cadence/configs/NDSS/config_japan_ppbr.json", "japan"),
        ("cmd/cadence/configs/NDSS/config_tdrive_ppbr.json", "tdrive")
    ]

    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_for_dataset, *dataset) for dataset in datasets]
        for future in futures:
            future.result()


def run_probabilistic_flooding():
    def run_for_dataset(config_file, dataset_name):
        run_values = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
        for buffer_size in buffer_size_values:
            for run in run_values:
                experiment_name = f"{dataset_name} - randomwalk-v1-random p_t=0.5 p_d=0.8 buffer={buffer_size} run={run}"
                messages_file_name = f"message_files/messages_{dataset_name}_type_{message_type}.json"

                modify_json(config_file, {
                    "simulation": {
                        "experiment_name": experiment_name,
                        "min_buffer_size": buffer_size,
                        "max_buffer_size": buffer_size,
                        "message_generation_type": message_type,
                        "messages_file": messages_file_name
                    }
                })
                command = f"./cmd/cadence/cadence sim {config_file}"
                run_command(command)

    datasets = [
        ("cmd/cadence/configs/NDSS/config_japan_randomwalk-v1-random.json", "japan"),
        ("cmd/cadence/configs/NDSS/config_tdrive_randomwalk-v1-random.json", "tdrive")
    ]

    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_for_dataset, *dataset) for dataset in datasets]
        for future in futures:
            future.result()


def run_handoff():
    def run_for_dataset(config_file, dataset_name):
        run_values = [1]
        for buffer_size in buffer_size_values:
            for run in run_values:
                experiment_name = f"{dataset_name} - randomwalk-v1 buffer={buffer_size} run={run}"
                messages_file_name = f"message_files/messages_{dataset_name}_type_{message_type}.json"

                modify_json(config_file, {
                    "simulation": {
                        "experiment_name": experiment_name,
                        "min_buffer_size": buffer_size,
                        "max_buffer_size": buffer_size,
                        "message_generation_type": message_type,
                        "messages_file": messages_file_name
                    }
                })
                command = f"./cmd/cadence/cadence sim {config_file}"
                run_command(command)

    datasets = [
        ("cmd/cadence/configs/NDSS/config_japan_randomwalk-v1.json", "japan"),
        ("cmd/cadence/configs/NDSS/config_tdrive_randomwalk-v1.json", "tdrive")
    ]

    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_for_dataset, *dataset) for dataset in datasets]
        for future in futures:
            future.result()


def run_max_flooding():
    def run_for_dataset(config_file, dataset_name):
        run_values = [1]
        for buffer_size in buffer_size_values:
            for run in run_values:
                experiment_name = f"{dataset_name} - broadcast buffer={buffer_size} run={run}"
                messages_file_name = f"message_files/messages_{dataset_name}_type_{message_type}.json"

                modify_json(config_file, {
                    "simulation": {
                        "experiment_name": experiment_name,
                        "min_buffer_size": buffer_size,
                        "max_buffer_size": buffer_size,
                        "message_generation_type": message_type,
                        "messages_file": messages_file_name
                    }
                })
                command = f"./cmd/cadence/cadence sim {config_file}"
                run_command(command)

    datasets = [
        ("cmd/cadence/configs/NDSS/config_japan_broadcast.json", "japan"),
        ("cmd/cadence/configs/NDSS/config_tdrive_broadcast.json", "tdrive")
    ]

    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_for_dataset, *dataset) for dataset in datasets]
        for future in futures:
            future.result()


def export_data():
    command = "sqlite3 -header -csv db/japan.db 'SELECT * FROM results_dbs' > results/raw-data/japan.csv"
    run_command(command)
    command = "sqlite3 -header -csv db/tdrive.db 'SELECT * FROM results_dbs' > results/raw-data/tdrive.csv"
    run_command(command)



def main():
    print("Starting experiments...")

    run_ppbr()
    run_mirage()
    run_probabilistic_flooding()
    run_handoff()
    run_max_flooding()

    export_data()

    print("All experiments executed successfully.")


if __name__ == "__main__":
    main()
