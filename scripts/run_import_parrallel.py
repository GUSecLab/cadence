import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor

def run_command(command):
    """Run a shell command and exit if it fails."""
    print(f"Executing: {command}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Error: Command failed - {command}")
        sys.exit(1)

def main():
    # Define the commands to execute
    commands = [
        "./cmd/cadence/cadence import cmd/cadence/configs/NDSS/config_japan_mirage.json",
        "./cmd/cadence/cadence import cmd/cadence/configs/NDSS/config_tdrive_mirage.json"
    ]

    # Run the commands in parallel
    with ThreadPoolExecutor() as executor:
        futures = [executor.submit(run_command, command) for command in commands]
        for future in futures:
            future.result()  # Wait for all commands to complete

    print("All commands executed successfully.")

if __name__ == "__main__":
    main()