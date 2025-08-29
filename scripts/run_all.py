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
        {"command": "python3 scripts/run_setup.py", "description": "Setting up the environment"},
        {"command": "python3 scripts/run_build.py", "description": "Building the simulator"},
        {"command": "python3 scripts/run_import_parrallel.py", "description": "Importing datasets"},
        {"command": "python3 scripts/run_experiments_parrallel.py", "description": "Running experiments"},
        {"command": "python3 scripts/run_plots.py", "description": "Generating plots"}
    ]

    # Execute each command sequentially
    for cmd in commands:
        run_command(cmd["command"], cmd["description"])

    print("All steps completed successfully.")

if __name__ == "__main__":
    main()