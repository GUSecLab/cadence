import os,time
import signal
import subprocess

# Find the process ID of the first Go script
pgrep_command = ['pgrep', '-f', './cadence']
pgrep_process = subprocess.Popen(pgrep_command, stdout=subprocess.PIPE)
pgrep_output = pgrep_process.communicate()[0]
first_script_pid = int(pgrep_output.strip())
# Wait for the first Go script to complete
while True:
    try:
        os.kill(first_script_pid, 0)
        print("still running, wait a little bit")
        time.sleep(5 * 60)
    except OSError:
        break

# Launch the second command
second_command_process = subprocess.Popen(['./cadence', 'config_capspotting.json', 'sim'])
output, errors = second_command_process.communicate()
with open("log_out.txt","w") as w:
	w.write("logout:\n")
	w.write(output.decode())
	w.write("\n-------------\n")
	w.write("errors:\n")
	w.write(errors.decode())
w.close()
print("congratulations! you earned your night")
