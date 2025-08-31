import subprocess

def run_command(command, description):
    """Run a shell command and print a warning if it fails."""
    print(f"Running: {description}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Warning: {description} failed, but continuing...")

def main():
    # List of commands to run
    commands = [
        {"command": "rm db/japan.db", "description": "Deleting old Japan database"},
        {"command": "rm db/tdrive.db", "description": "Deleting old TDrive database"},
        {"command": "rm -rf results/raw-data/* results/raw-data/.[!.]*", "description": "Clearing old data but keeping directory"},
        {"command": "rm -rf results/plots/*", "description": "Clearing old plots"},
        {"command": "rm -rf message_files/*", "description": "Clearing old message files"},
        {"command": "rm cmd/cadence/cadence", "description": "Removing old executable"},
    ]

    # Execute each command sequentially
    for cmd in commands:
        run_command(cmd["command"], cmd["description"])

    print("All steps completed (with possible warnings).")

if __name__ == "__main__":
    main()