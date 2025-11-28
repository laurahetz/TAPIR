##### PARSE result CSV


import sys
import pandas as pd
import numpy as np

input_file=sys.argv[1]

# Read the CSV file
df = pd.read_csv(input_file, delimiter=',')  # Adjust delimiter if needed
df = df.replace(np.nan, "None")

# Get unique combinations of PIR and VC
unique_combinations = df[['pir_type', 'vc_type', 'db_size','rec_size']].drop_duplicates()
print(unique_combinations)

# Generate output filename
file_path = 'averages.csv'
averages_df = pd.DataFrame()

for index, row in unique_combinations.iterrows():
    PIR_value = row['pir_type']
    VC_value = row['vc_type']
    DB_size = row['db_size']
    Rec_size = row['rec_size']
    
    # Filter dataframe for the current combination
    filtered_df = df[(df['pir_type'] == PIR_value) & (df['vc_type'] == VC_value) & (df['db_size'] == DB_size)& (df['rec_size'] == Rec_size)]

    exclude_cols = ['pir_type', 'vc_type', 'repetition', 'rec_size']
    averages = filtered_df.drop(columns=exclude_cols).mean()

    avg_row = pd.DataFrame([averages], index=[PIR_value + "_" + VC_value + "_" + str(Rec_size)])
    
    # Print averages to new CSV file
    averages_df = pd.concat([averages_df, avg_row])

for key in averages_df.index.unique():
    print(key)
    partition = averages_df.loc[[key]] # Use double brackets to keep it as a DataFrame
    partition.to_csv(f'{key}.csv', index=False)