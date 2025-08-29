import os
import subprocess
import glob

def run_command(command, description):
    """Run a shell command and exit if it fails."""
    print(f"Running: {description}")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"Error: {description} failed.")
        exit(1)

def main():
    # Ensure the results directory exists
    results_dir = "results/plots"
    os.makedirs(results_dir, exist_ok=True)

    # Process each CSV file in the data directory
    for csv_file in glob.glob("results/raw-data/*.csv"):
        base_name = os.path.splitext(os.path.basename(csv_file))[0]
        output_file = os.path.join(results_dir, f"{base_name}.pdf")
        
        # Run the make-graphs.py script
        command = f"python scripts/make-graphs.py --data-file \"{csv_file}\" --output-file \"{output_file}\" --metrics nl al dr custom custom2 lat"
        run_command(command, f"Generating graph for {csv_file}")

    # Crop each PDF in the results directory
    for pdf_file in glob.glob(f"{results_dir}/*.pdf"):
        command = f"pdfcrop \"{pdf_file}\" \"{pdf_file}\""
        run_command(command, f"Cropping PDF {pdf_file}")

    command = "python scripts/run_make_timely_results.py"
    run_command(command, "Running make timely results script")

    print("All graphs generated and cropped successfully.")

if __name__ == "__main__":
    main()