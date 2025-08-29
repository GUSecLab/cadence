import os
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
    # Navigate to the cmd/cadence directory
    cadence_dir = os.path.join(os.getcwd(), "cmd", "cadence")
    if not os.path.exists(cadence_dir):
        print(f"Error: Directory {cadence_dir} does not exist.")
        sys.exit(1)

    os.chdir(cadence_dir)
    print(f"Changed directory to {cadence_dir}")

    # Run go build
    run_command("go build", "Building Go project")

    print("Go build completed successfully.")

if __name__ == "__main__":
    main()