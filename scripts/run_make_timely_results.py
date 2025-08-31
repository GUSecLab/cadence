import subprocess
import sys

def run_command(command, description):
    """Run a shell command and exit if it fails."""
    print(f"Running: {description}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Error: {description} failed.")
        sys.exit(1)

def main():
    # List of commands to run
    commands = [
        {"command": "python3 scripts/export_timely_data.py", "description": "Exporting timely data"},
        {"command": "python3 scripts/make-timely-graphs.py", "description": "Making timely graphs"}
    ]

    # Execute each command sequentially
    for cmd in commands:
        run_command(cmd["command"], cmd["description"])

    print("All steps completed successfully.")

if __name__ == "__main__":
    main()