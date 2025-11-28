import pandas as pd
import numpy as np
import sys
import glob
import os
import re

def process_csv(input_file):
    # Read the CSV file with headers
    df = pd.read_csv(input_file)
    print(f"\nProcessing {input_file}")
    print("Raw values:")
    print(f"db_size: {df['db_size'].values[0]}")
    print(f"BW_Digests: {df['BW_Digests'].values[0]}")
    print(f"BW_HintReqs: {df['BW_HintReqs'].values[0]}")
    print(f"BW_HintResps: {df['BW_HintResps'].values[0]}")
    print(f"RT_GenDigest: {df['RT_GenDigest'].values[0]}")
    
    # Database size (log2)
    db_size = np.log2(df['db_size'])
    
    # Get record size from filename
    basename = os.path.basename(input_file)
    parts = basename.split('_')
    rec_size = int(parts[3].replace('.csv', ''))

    # Offline Bandwidth (bytes to KB)
    offline_bw = (df['BW_Digests'] + df['BW_HintReqs'] + df['BW_HintResps']) / 1024.0
    
    # Offline RT (1 time) (ns to ms)
    offline_rt_one_time = df['RT_GenDigest'] / 1_000_000.0
    
    # Offline RT (per client) (ns to ms)
    offline_rt_per_client = (df['RT_RequestHint'] + df['RT_GenHint'] + df['RT_VerSetup']) / 1_000_000.0

    # Online Bandwidth (bytes to KB)
    online_bw = (df['BW_Queries'] + df['BW_Answers']) / 1024.0
    
    # Online RT (ns to ms)
    online_rt_per_client = (df['RT_Query'] + df['RT_Answer'] + df['RT_Reconstruct']) / 1_000_000.0

    print("\nProcessed values:")
    print(f"db_size (log2): {db_size.values[0]}")
    print(f"offline_bw (KB): {offline_bw.values[0]}")
    print(f"offline_rt_one_time (s): {offline_rt_one_time.values[0]}")
    print(f"online_bw (KB): {online_bw.values[0]}")
    print(f"online_rt (ms): {online_rt_per_client.values[0]}")

    # Extract scheme name from filename
    scheme = os.path.basename(input_file).replace('.csv', '')

    # Create new DataFrame with all columns
    result_df = pd.DataFrame({
        'scheme': scheme,
        'db_size': db_size,
        'rec_size': rec_size,  # Now using rec_size from filename
        'offline_bw': offline_bw,
        'offline_rt_one_time': offline_rt_one_time,
        'offline_rt_per_client': offline_rt_per_client,
        'online_bw': online_bw,
        'online_rt_per_client': online_rt_per_client
    }).round(2)
    return result_df

def format_row(row):
    # Format each value with \qty{} if it exceeds 999.99
    scheme_name = format_scheme_name(row['scheme'])
    db_size = f"\\multirow{{1}}{{*}}{{$2^{{{int(row['db_size'])}}}$}}"  # Fixed LaTeX formatting
    
    # For DPF scheme, show "-" for offline columns
    if "dpfmac" in scheme_name.lower():
        offline_bw = "-"
        offline_rt_one_time = "-"
        offline_rt_per_client = "-"
    else:
        offline_bw = f"\\qty{{{row['offline_bw']:.2f}}}{{}}" if row['offline_bw'] > 999.99 else f"{row['offline_bw']:.2f}"
        # Changed from s to ms, no need to multiply here since conversion done in process_csv
        offline_rt_one_time = f"\\qty{{{row['offline_rt_one_time']:.2f}}}{{}}" if row['offline_rt_one_time'] > 999.99 else f"{row['offline_rt_one_time']:.2f}"
        offline_rt_per_client = f"\\qty{{{row['offline_rt_per_client']:.2f}}}{{}}" if row['offline_rt_per_client'] > 999.99 else f"{row['offline_rt_per_client']:.2f}"
    
    online_bw = f"\\qty{{{row['online_bw']:.2f}}}{{}}" if row['online_bw'] > 999.99 else f"{row['online_bw']:.2f}"
    online_rt_per_client = f"\\qty{{{row['online_rt_per_client']:.2f}}}{{}}" if row['online_rt_per_client'] > 999.99 else f"{row['online_rt_per_client']:.2f}"
    return f"{db_size} & {scheme_name} & {offline_bw} & {offline_rt_one_time} & {offline_rt_per_client} & {online_bw} & {online_rt_per_client} \\\\\n"

def format_scheme_name(filename):
    # Extract components from filename
    parts = filename.split('_')
    if len(parts) < 4:
        return filename
    
    scheme_type = parts[0]
    scheme = parts[1]
    proof = parts[2]
    size = parts[3]
    
    # Format scheme name based on components
    if scheme == "DPF128":
        return "\\dpfmac"
    elif scheme == "SinglePass":
        return "\\singlepass"
    elif scheme == "Matrix":
        if proof == "MerkleTree":
            return "\\linmt"
        elif proof == "PointProof":
            return "\\linpp"
    elif scheme == "TAPIR":
        if proof == "MerkleTree":
            return "\\textbf{\\tapirmt}"
        elif proof == "PointProof":
            return "\\textbf{\\tapirpp}"
    return filename

def get_record_size_from_filename(filename):
    # Extract size from patterns like 'PIR_SinglePass_None_32.csv'
    basename = os.path.basename(filename)
    parts = basename.split('_')
    
    print(f"Trying to extract record size from: {basename}")
    if len(parts) >= 4:
        try:
            size = int(parts[3].replace('.csv', ''))
            print(f"Found record size: {size}")
            return size
        except ValueError:
            pass
    
    print(f"No record size found in filename: {basename}")
    return None

def create_latex_table(rows, record_size):
    # First, count how many rows have each N value
    n_counts = {}
    for row in rows:
        match = re.search(r'\$2\^{(\d+)}\$', row)
        if match:
            n = int(match.group(1))
            n_counts[n] = n_counts.get(n, 0) + 1

    table = "\\begin{tabular}{@{}cl|rrr|rr@{}}\n"
    table += "\\toprule\n"
    table += "\\multirow{2}{*}{N} & \n"
    table += "      \\multirow{2}{*}{PIR} & \n"
    table += "      \\multicolumn{3}{c|}{\\bf Offline} & \n"
    table += "      \\multicolumn{2}{c}{\\bf Online} \\\\\n"
    table += " & & \n"
    table += "        \\multicolumn{1}{c}{\\makecell[c]{BW {[kiB]}}} & \n"
    table += "        \\multicolumn{1}{c}{\\makecell[c]{RT {[ms]}\\\\(1-Time)}} & \n"
    table += "        \\multicolumn{1}{c|}{\\makecell[c]{RT {[ms]}\\\\(Per-Client)}} & \n"
    table += "        \\multicolumn{1}{c}{\\makecell[c]{BW {[kiB]}}} & \n"
    table += "        \\multicolumn{1}{c}{\\makecell[c]{RT {[ms]}}} \\\\\n"
    table += "\\midrule\n"
    
    # Add rows with proper multirow counts
    current_n = None
    rows_since_n_change = 0
    for row in rows:
        match = re.search(r'\$2\^{(\d+)}\$', row)
        if match:
            n = int(match.group(1))
            if current_n is not None and n != current_n:
                table += "    \\midrule\n"
                rows_since_n_change = 0
            current_n = n
            
            # Replace the N column formatting using raw strings
            if rows_since_n_change == 0:
                # First row of this N value - add multirow with count
                row = re.sub(r'\\multirow\{1\}\{\*\}\{\$2\^\{(\d+)\}\$\}', 
                           rf'\\multirow{{{n_counts[n]}}}{{*}}{{$2^{{{n}}}$}}', 
                           row)
            else:
                # Subsequent rows - empty N column
                row = re.sub(r'\\multirow\{1\}\{\*\}\{\$2\^\{(\d+)\}\$\}', 
                           '', 
                           row)
            rows_since_n_change += 1
        table += row

    table += "\\bottomrule\n"
    table += "\\end{tabular}\n"
    return table

def get_scheme_order(scheme_name):
    # Define order: DPF, Matrix-MT, Matrix-PP, TAPIR-MT, TAPIR-PP, Singlepass
    order = {
        "\\dpfmac": 0,
        "\\linmt": 1,
        "\\linpp": 2,
        "\\textbf{\\tapirmt}": 3,
        "\\textbf{\\tapirpp}": 4,
        "\\singlepass": 5
    }
    return order.get(scheme_name, 999)  # Unknown schemes go last

if __name__ == "__main__":
    # Usage: python make-table.py apir/ output_prefix
    input_dir = sys.argv[1]
    output_prefix = sys.argv[2]

    print(f"Looking for CSV files in: {input_dir}")

    # Dictionary to store rows by record size
    rows_by_recsize = {}

    # Process all CSV files
    csv_files = sorted(glob.glob(os.path.join(input_dir, "*.csv")))
    print(f"Found {len(csv_files)} CSV files")
    print("CSV files found:", csv_files)  # Print actual filenames found
    
    for csv_file in csv_files:
        print(f"Processing file: {csv_file}")
        record_size = get_record_size_from_filename(csv_file)
        if record_size is None:
            print(f"Warning: Could not extract record size from {csv_file}")
            continue
            
        print(f"Found record size: {record_size}")
        df = process_csv(csv_file)
        for _, row in df.iterrows():
            if record_size not in rows_by_recsize:
                rows_by_recsize[record_size] = []
            rows_by_recsize[record_size].append(format_row(row))

    print(f"\nFound record sizes: {list(rows_by_recsize.keys())}")
    
    # Create separate files for each record size
    for rec_size, rows in rows_by_recsize.items():
        # Sort rows first by N, then by scheme order
        def get_n_from_latex(row_str):
            # Extract N from "\multirow{1}{*}{$2^{10}$} & ..."
            match = re.search(r'\$2\^{(\d+)}\$', row_str)
            return int(match.group(1)) if match else 0
            
        sorted_rows = sorted(rows, key=lambda x: (
            get_n_from_latex(x),  # Get N value
            get_scheme_order(x.split('&')[1].strip())  # Get scheme name
        ))
        
        output_file = f"{output_prefix}-{rec_size}.tex"
        print(f"Creating output file: {output_file}")
        latex_table = create_latex_table(sorted_rows, rec_size)
        with open(output_file, 'w') as f:
            f.write(latex_table)
        print(f"Created table for record size {rec_size} with {len(rows)} rows")

    print("\nDone!")