##### PARSE update result CSV

import sys
import pandas as pd
import numpy as np

if len(sys.argv) != 2:
    print("Usage: python parse-update-csv.py <input_csv_file>")
    sys.exit(1)

input_file = sys.argv[1]

# Read the CSV file
df = pd.read_csv(input_file, delimiter=',')  # Adjust delimiter if needed
df = df.replace(np.nan, "None")

# Get unique combinations of PIR, VC, db_size, and rec_size
unique_combinations = df[['pir_type', 'vc_type', 'db_size', 'rec_size','update_type','num_updates']].drop_duplicates()
print("Unique combinations found:")
print(unique_combinations)

# Generate output filename
averages_df = pd.DataFrame()

for index, row in unique_combinations.iterrows():
    PIR_value = row['pir_type']
    VC_value = row['vc_type']
    DB_size = row['db_size']
    Rec_size = row['rec_size']
    Up_type = row['update_type']
    Num_up = row['num_updates']
    
    # print(f"\nProcessing: {PIR_value}_{VC_value}_{DB_size}_{Rec_size}_{Up_type}_{Num_up}")
    
    # Filter dataframe for the current combination
    filtered_df = df[
        (df['pir_type'] == PIR_value) & 
        (df['vc_type'] == VC_value) & 
        (df['db_size'] == DB_size) & 
        (df['rec_size'] == Rec_size) &
        (df['update_type'] == Up_type) &
        (df['num_updates'] == Num_up) 
    ]
    
    if filtered_df.empty:
        print(f"No data found for combination: {PIR_value}_{VC_value}_{DB_size}_{Rec_size}_{Up_type}_{Num_up}")
        continue
    
    # Exclude non-numeric columns from averaging
    exclude_cols = ['pir_type', 'vc_type', 'repetition', 'rec_size', 'update_type','num_updates', 'db_size', 'part_size']
    # Only keep columns that exist in the dataframe
    exclude_cols = [col for col in exclude_cols if col in filtered_df.columns]
    
    # Calculate averages for numeric columns only
    numeric_df = filtered_df.drop(columns=exclude_cols).select_dtypes(include=[np.number])
    
    # Drop columns that start with BW_ or RT_ but don't contain "Update"
    cols_to_drop = []
    for col in numeric_df.columns:
        if (col.startswith('BW_') or col.startswith('RT_')) and 'Update' not in col:
            cols_to_drop.append(col)
    
    numeric_df = numeric_df.drop(columns=cols_to_drop)
    
    if numeric_df.empty:
        print(f"No numeric columns found for averaging")
        continue
        
    averages = numeric_df.mean()
    
    # Create amortized columns by dividing update-related columns by num_updates
    amortized_averages = averages.copy()
    for col in averages.index:
        if col.startswith('BW_') and 'Update' in col:
            amortized_col_name = col.replace('BW_', 'BW_Amortized_')
            amortized_averages[amortized_col_name] = averages[col] / Num_up
        elif col.startswith('RT_') and 'Update' in col:
            amortized_col_name = col.replace('RT_', 'RT_Amortized_')
            amortized_averages[amortized_col_name] = averages[col] / Num_up
    
    # Create row with meaningful index name (excluding db_size from filename)
    index_name = f"{PIR_value}_{VC_value}_{Rec_size}_{Up_type}_{Num_up}"
    avg_row = pd.DataFrame([amortized_averages], index=[index_name])
    
    # Add the categorical columns back at the beginning
    avg_row.insert(0, 'pir_type', PIR_value)
    avg_row.insert(1, 'vc_type', VC_value)
    avg_row.insert(2, 'db_size', DB_size)
    avg_row.insert(3, 'rec_size', Rec_size)
    avg_row.insert(4, 'update_type', Up_type)
    avg_row.insert(5, 'num_updates', Num_up)
    
    #print(f"Averaged {len(filtered_df)} rows")
    
    # Concatenate to main dataframe
    averages_df = pd.concat([averages_df, avg_row])

# Save individual CSV files for each unique combination
print(f"\nSaving {len(averages_df.index.unique())} result files...")

for key in averages_df.index.unique():
    print(f"Saving: {key}.csv")
    partition = averages_df.loc[[key]]  # Use double brackets to keep it as a DataFrame
    
    # Round numeric columns to 2 decimal places (excluding the categorical columns)
    numeric_cols = partition.select_dtypes(include=[np.number]).columns
    partition[numeric_cols] = partition[numeric_cols].round(2)
    
    partition.to_csv(f'{key}.csv', index=False)

# Also save a combined averages file
averages_df_rounded = averages_df.copy()
numeric_cols = averages_df_rounded.select_dtypes(include=[np.number]).columns
averages_df_rounded[numeric_cols] = averages_df_rounded[numeric_cols].round(2)
averages_df_rounded.to_csv('combined_averages.csv', index=True)

print(f"\nCompleted! Processed {len(unique_combinations)} unique combinations.")
print("Individual files saved as: <pir_type>_<vc_type>_<rec_size>_<update_type>_<num_updates>.csv")
print("Combined results saved as: combined_averages.csv")