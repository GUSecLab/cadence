import subprocess
import sys


def run_command(command):
    """Run a shell command and exit if it fails."""
    print(f"Executing: {command}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Error: Command failed - {command}")
        # sys.exit(1)

def export_data():   
    command = "sqlite3 -header -csv db/japan.db 'SELECT * FROM bandwidths' > results/raw-data/japan_bandwidths.csv"
    run_command(command)
    command = "sqlite3 -header -csv db/tdrive.db 'SELECT * FROM bandwidths' > results/raw-data/tdrive_bandwidths.csv"
    run_command(command)
    command = "sqlite3 -header -csv db/japan.db 'SELECT * FROM delivered_message_dbs' > results/raw-data/japan_delivered_messages.csv"
    run_command(command)
    command = "sqlite3 -header -csv db/tdrive.db 'SELECT * FROM delivered_message_dbs' > results/raw-data/tdrive_delivered_messages.csv"
    run_command(command)

def main():
    print("starting export of timely data...")
    export_data()

if __name__ == "__main__":
    main()
