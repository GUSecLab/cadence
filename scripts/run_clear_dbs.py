import sqlite3
import os

def clear_database_except(db_path, tables_to_keep):
    """
    Clears all tables in the SQLite database except the specified ones.
    If the database file does not exist, skip and print a message.

    :param db_path: Path to the SQLite database file.
    :param tables_to_keep: List of table names to keep.
    """
    if not os.path.exists(db_path):
        print(f"Database not found, skipping: {db_path}")
        return

    try:
        # Connect to the SQLite database
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()

        # Get the list of all tables in the database
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
        all_tables = [row[0] for row in cursor.fetchall()]

        # Determine which tables to clear
        tables_to_clear = [table for table in all_tables if table not in tables_to_keep]

        # Clear each table
        for table in tables_to_clear:
            print(f"Clearing table: {table} in {db_path}")
            cursor.execute(f"DELETE FROM {table};")
            conn.commit()

        print(f"Database cleared successfully (except specified tables): {db_path}")

    except sqlite3.Error as e:
        print(f"An error occurred with {db_path}: {e}")
    finally:
        if 'conn' in locals():
            conn.close()

if __name__ == "__main__":
    # Tables to keep
    print("clearing databases")
    tables_to_keep = ["dataset_encounter_enumerations", "datasets", "encountered_nodes", "encounters", "events"]  

    # Clear the databases if they exist
    clear_database_except("db/japan.db", tables_to_keep)
    clear_database_except("db/tdrive.db", tables_to_keep)
