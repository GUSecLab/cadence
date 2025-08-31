# YJMOB100K human mobility dataset

https://www.nature.com/articles/s41597-024-03237-9
https://zenodo.org/records/10836269

YJMOB100k (Yahoo Japan, Mobility, 100,000 individuals) is an open-source and anonymized human mobility dataset tracking the mobiliy trajectories of 100,000 individuals. 

trajectories are discretized into 500 x 500 meter cells grouped into a 200 x 200 cell area and time-discretized into 30 minute intervals. 

Dataset 1 tracks mobility over a 75 day normal business period. Dataset 2 tracks mobility over 75 days, with 60 days of normality and 15 days of an emergency situation. 

POI dataset identifies points of interest such as restaurants and stores, and the POI category lists names the categories. 

Read more in the official data descriptor: 
https://www.nature.com/articles/s41597-024-03237-9

## Usage: 

From data descriptor:

"Since the dataset is de-identified and anonymized through the redaction of the actual date and location coordinates, the Human Research Protection Program in the Institutional Review Board (IRB) at New York University determined that the data does not meet the federal regulations definition of human subject, and therefore, it is not under the purview of the IRB."

To extract dataset:

`$unzip YJMOB100K.zip`

this will produce four .csv.gz files. To unzip the .gz files:

`$gzip -d *.gz`

### Recommended Directory Structure

this is the recommended directory structure to store csv data files

make sure the dataset file path in the config file matches directory structure

/japan

----/data

--------/categories

------------cell_POIcat.csv

------------POI_datacategories.csv

--------/dataset_1

------------yjmob100k-dataset1.csv

--------/dataset_2

------------yjmob100k-dataset2.csv

## Notes

### File Size

the fully uncompressed data files are quite large, totally around 3 GB in size. It is recommended to store the files in compressed form and uncompress only for data import. Problems may occur even with using git-lfs. 

### Location System

The dataset discretizes location data into a 200 x 200 grid of 500m x 500m blocks. Exact location of the trajectory within a 500m x 500m block is not available. 

### mini version 

The dataset_mini dataset is a miniature version of the japan dataset. It was created for resource efficiency when running experiments repeatedly. This dataset only contains data from a sample of 500 users from the original dataset (dataset_1). 