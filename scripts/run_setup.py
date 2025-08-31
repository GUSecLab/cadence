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

def install_go():
    """Install Go if not already installed."""
    print("Checking for Go installation...")
    result = subprocess.run("go version", shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if result.returncode != 0:
        print("Go is not installed. Installing Go...")
        run_command(
            "wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz -O /tmp/go.tar.gz && "
            "sudo rm -rf /usr/local/go && "
            "sudo tar -C /usr/local -xzf /tmp/go.tar.gz && "
            "rm /tmp/go.tar.gz && "
            "export PATH=$PATH:/usr/local/go/bin",
            "Installing Go"
        )
    else:
        print("Go is already installed.")

def setup_go_modules():
    """Set up Go modules and dependencies."""
    print("Setting up Go modules...")
    # Path to the subdirectory containing the go.mod file
    go_mod_dir = os.path.join(os.getcwd(), "path/to/subdirectory")  # Replace with actual path

    if not os.path.exists(os.path.join(go_mod_dir, "go.mod")):
        print(f"No go.mod file found in {go_mod_dir}. Skipping Go module setup.")
        return

    original_dir = os.getcwd()
    os.chdir(go_mod_dir)

    try:
        run_command("go mod tidy", "Resolving Go dependencies")
    finally:
        os.chdir(original_dir)

def install_python_dependencies():
    """Install Python dependencies."""
    print("Installing Python dependencies...")
    requirements_file = os.path.join(os.getcwd(), "scripts", "requirements.txt")
    if os.path.exists(requirements_file):
        run_command(f"pip install -r {requirements_file}", "Installing Python dependencies")
    else:
        print(f"No requirements.txt file found at {requirements_file}. Skipping Python dependency installation.")

def install_sqlite3():
    """Install SQLite3 if not already installed."""
    print("Checking for SQLite3 installation...")
    result = subprocess.run("sqlite3 --version", shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if result.returncode != 0:
        print("SQLite3 is not installed. Installing SQLite3...")
        run_command("sudo apt update && sudo apt install -y sqlite3", "Installing SQLite3")
    else:
        print("SQLite3 is already installed.")

def install_latex():
    """Install LaTeX packages required for plotting."""
    print("Checking for LaTeX installation...")
    result = subprocess.run("pdflatex --version", shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if result.returncode != 0:
        print("LaTeX is not installed. Installing LaTeX packages...")
        run_command("sudo apt install -y texlive-latex-base", "Installing LaTeX base")
        run_command("sudo apt install -y texlive-latex-extra dvipng cm-super", "Installing LaTeX extra tools for plotting")
        run_command("sudo apt install -y texlive-extra-utils", "Installing pdfcrop and other utilities")
    else:
        print("LaTeX is already installed.")

def create_results_directories():
    """Create results/raw-data and results/plots directories if they don't exist."""
    print("Ensuring results directories exist...")
    base_dir = os.path.join(os.getcwd(), "results")
    raw_data_dir = os.path.join(base_dir, "raw-data")
    plots_dir = os.path.join(base_dir, "plots")

    for directory in [base_dir, raw_data_dir, plots_dir]:
        if not os.path.exists(directory):
            os.makedirs(directory)
            print(f"Created: {directory}")
        else:
            print(f"Exists: {directory}")

def create_db_directories():
    """Create db directories if they don't exist."""
    print("Ensuring db directories exist...")
    db_dir = os.path.join(os.getcwd(), "db")

    if not os.path.exists(db_dir):
        os.makedirs(db_dir)
        print(f"Created: {db_dir}")
    else:
        print(f"Exists: {db_dir}")

def main():
    print("Setting up the environment for running Go code...")

    # Step 1: Install Go
    install_go()

    # Step 2: Set up Go modules
    setup_go_modules()

    # Step 3: Install Python dependencies
    install_python_dependencies()

    # Step 4: Install SQLite3
    install_sqlite3()

    # Step 5: Install LaTeX packages for plotting
    install_latex()

    # Step 6: Create results/raw-data and results/plots directories
    create_results_directories()
    
    # Step 7: Create db directory
    create_db_directories()

    print("Environment setup complete. You can now run your Go code.")

if __name__ == "__main__":
    main()
