import os
import shutil

def clear_results():
    base_dir = "results"
    subdirs = ["raw-data", "plots"]

    # If results/ does not exist
    if not os.path.exists(base_dir):
        print(f"Base directory does not exist, skipping: {base_dir}")
        return

    for sub in subdirs:
        sub_path = os.path.join(base_dir, sub)

        # If subdirectory does not exist
        if not os.path.exists(sub_path):
            print(f"Directory does not exist, skipping: {sub_path}")
            continue

        items = os.listdir(sub_path)

        # If subdirectory is empty
        if not items:
            print(f"Directory is already empty, skipping: {sub_path}")
            continue

        # Otherwise, clear its contents
        for item in items:
            item_path = os.path.join(sub_path, item)
            if os.path.isfile(item_path) or os.path.islink(item_path):
                os.remove(item_path)
            elif os.path.isdir(item_path):
                shutil.rmtree(item_path)

        print(f"Cleared contents of: {sub_path}")

if __name__ == "__main__":
    print("clearing results")
    clear_results()
