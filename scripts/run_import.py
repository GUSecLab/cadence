import subprocess
import sys

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

    # Execute each command sequentially
    for command in commands:
        run_command(command)

    print("All commands executed successfully.")

if __name__ == "__main__":
    main()