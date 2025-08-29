
-- @block
SELECT * FROM bandwidths;

-- @block
SELECT * FROM buffer_maxes;

-- @block
SELECT * FROM dataset_encounter_enumerations;

-- @block
SELECT * FROM datasets;

-- @block
SELECT * FROM delivered_message_dbs;

-- @block
SELECT * FROM encountered_nodes;

-- @blocks
SELECT * FROM encounters;

-- @block
SELECT * FROM epoch_loads;

-- @block
SELECT * FROM events;


-- @block
SELECT * 
FROM encounters 
WHERE dataset_name = 'japan'
ORDER BY time ASC
LIMIT 10;

-- @block
SELECT COUNT(*) AS encounter_count
FROM encounters
WHERE PPBR = 1;

-- @block
SELECT * FROM experiment_configs;

-- @block
SELECT * FROM experiment_families;

-- @block
SELECT * FROM experiments;

-- @block
SELECT * FROM message_dbs;

-- @block
SELECT * FROM node_lists;

-- @block
SELECT * FROM results_dbs;

-- @block 
SELECT * FROM pmg_edges;

-- @block
SELECT MAX(time) latest_time FROM events WHERE dataset_name = 'geolife';
SELECT MIN(time) earliest_time FROM events WHERE dataset_name = 'geolife';

-- @block
SELECT * FROM message_dbs LIMIT 10;

-- @block
SELECT * FROM events WHERE dataset_name = 'japan' LIMIT 100;

-- @block
SELECT * FROM events WHERE dataset_name = 'cabspotting2D' LIMIT 10;

-- @block
SELECT * FROM events WHERE dataset_name = 'japan dataset 1 mini' LIMIT 10;

-- @block
SELECT * FROM bandwidths WHERE experiment_name = 'cabspotting - addressing dp run 1 k=8';

-- @block
SELECT * FROM district_tables;

-- @block 
SELECT * FROM node_regions WHERE dataset = 'cabspotting2D';

-- @block
SELECT * FROM encounters WHERE experiment_name = 'cabspotting - addressing dp run 1 k=8';

-- gets the amount of distinct nodes
-- @block 
SELECT COUNT(DISTINCT node1) AS distinct_node_count
FROM encounters
WHERE dataset_name = 'cabspotting2D';


-- @block 
ALTER TABLE districts DROP COLUMN experiment_name;

-- @block 
DROP TABLE IF EXISTS districts;

-- @block 
DROP TABLE IF EXISTS dataset_encounter_enumerations;

-- @block
DROP TABLE IF EXISTS encounters;

-- @block 
SELECT 
    MIN(time) AS smallest_time, 
    MAX(time) AS largest_time
FROM events;

-- @block 
SELECT * FROM districts;

-- Replace 'your_experiment_name' with the actual experiment name
-- @block
SELECT COUNT(*) AS delivered_message_count
FROM delivered_message_dbs
WHERE experiment_name = 'japan - ppbr run=1';

-- @block
SELECT COUNT(DISTINCT message_id) AS message_count
FROM message_dbs
WHERE experiment_name = 'japan - ppbr run=1';

-- @block 
START TRANSACTION;

SET @exp_name = 'tdrive - broadcast run=2';

DELETE FROM bandwidths WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM buffer_maxes WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM delivered_message_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM encountered_nodes WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
-- DELETE FROM encounters WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM epoch_loads WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiment_configs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiment_families WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiments WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM message_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM results_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;

COMMIT;

-- @block 
START TRANSACTION;

SET @exp_name = 'cabspotting - ppbr run=1';

DELETE FROM bandwidths WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM buffer_maxes WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM delivered_message_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM encountered_nodes WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM epoch_loads WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiment_configs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiment_families WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM experiments WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM message_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM results_dbs WHERE experiment_name = CONVERT(@exp_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;

COMMIT;

-- destroy district table and node regions tables 
-- @block 
DROP TABLE IF EXISTS district_tables;
DROP TABLE IF EXISTS node_regions;

-- @block 
SELECT * FROM district_tables;

-- @block
SELECT * FROM node_regions;

-- @block
-- fully clearing everything including dataset and encounters table
SET @dataset_name = 'wipha natural disaster';

DELETE FROM dataset_encounter_enumerations WHERE dataset_name = CONVERT(@dataset_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM encounters WHERE dataset_name = CONVERT(@dataset_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM node_lists WHERE dataset_name = CONVERT(@dataset_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM datasets WHERE dataset_name = CONVERT(@dataset_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;
DELETE FROM events WHERE dataset_name = CONVERT(@dataset_name USING utf8mb4) COLLATE utf8mb4_unicode_ci;

