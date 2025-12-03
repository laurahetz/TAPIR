# TwoServerAPIR

This is a prototype implementation of TAPIR, Two-server Authenticated Private Information Retrieval.
Check out our [paper](https://ia.cr/2025/2177) (to appear at ACNS'25) for details.

> :warning: **Disclaimer**: This code is provided as an experimental implementation for testing purposes and should not be used in a productive environment. We cannot guarantee security and correctness.


## Module organization (so far)

**tapir/** 
- **app**: 
    - Contains `json` files specifying the executed benchmarks.
    - Will contain the results of the benchmarks specified in the config files.
- **benchmark**: Benchmarking suite. Will read config files, execute the described benchmarks, and save the results in `csv`.
- **container**: Container description.
- **eval**: Evaluation scripts for the benchmarking results
- **modules/**
    - **database**: Static database implementation based on SinglePass/Checklist with extention to support updates.
    - **dpf-go/**: Optimized Distributed Point Functions from [dpf-go](https://github.com/dimakogan/dpf-go/). 
    - **libfss/**: Less-Optimized general Function Secret Sharing implementation from [libfss](https://github.com/frankw2/libfss).
    - **psetggm/**: Adapted C++ code from [SinglePass](https://github.com/SinglePass712/Submission), includes C++ permutation code.
    - **pp/**: Adapted PointProof implementation from [yacovm/PoL](https://github.com/yacovm/PoL).
    - **merkle/**: Adapted Merkle Tree implementation from [apir-code](https://github.com/dedis/apir-code).
    - **utils/**: Common functions, incl. utils for randomness.
    - **vc/**: Vector commitment schemes and interface for their generic use, including MerkleTrees (see `merkle/`) and PointProofs (see `pp/`).
- **pir/**: Two-server (A)PIR schemes and a generic interface definition for these based on our paper's API.

## Protocol Types

This repository implements multiple different (A)PIR and vector commitment (VC) schemes and allows their benchmarking.

### PIR Types

0. `pir.PIR_Matrix`: Linear PIR scheme with $\sqrt{|DB|}$ rebalancing optimization based on the original PIR paper of Chor, Goldreich, Kushilevitz, and Sudan. Defined in `pir/pir_matrix.go`.
1. `pir.PIR_DPF` Unauthenticate DPF PIR with 1 bit outputs. Defined in `pir/pir_dpf.go`.
2. `pir.PIR_SinglePass`: SinglePass PIR adapted to actually measure bandwidth and to reduce bandwidth in offline phase. Defined in `pir/pir_singlepass.go`.
3. `pir.APIR_DPF128`: DPF-based authenticated PIR for 128 bit field. Defined in `pir/apir_dpf128.go`.
4. `pir.APIR_PARALLEL_DPF`: DPF-based APIR for retrieving one record in each of Q database partitions. Used in `APIR_TAPIR`. Defined in `pir/apir_parallel_dpf.go`.
5. `pir.APIR_TAPIR`: Our Two-Server Authenticated PIR protocol. Defined in `pir/apir_tapir.go`.
6. `pir.APIR_Matrix`: Authenticated version of `pir.PIR_Matrix` using VC. Defined in `pir/apir_matrix.go`.

### VC Types

0. `vc.None`: No vector commitment is used. This is used for unauthenticated schemes and authenticated DPF schemes with MAC.
1. `vc.VC_PointProof`: PointProofs (see `modules/pp/`).
2. `vc.VC_MerkleTree`: MerkleTree (see `modules/merkle/`).


## Requirements

- Golang 1.23
- [osu-crypto/libOTe](https://github.com/osu-crypto/libOTe/tree/master)
- podman (or an equivalent container orchestration software)
- make


## Containerized Environment

This repository contains a containerized build and run environment for our protocol benchmarks.
This environment simplifies running TAPIR and reproducing our benchmarking results.

The provided `Makefile` contains all required commands. 
While the following commands use *Podman*, a container orchestration software, compatible software, e.g. Docker, can be used instead.

To build and run benchmarks, the following steps are needed (from the root of this directory): 

1. Ensure a config file with the desired benchmarks exists at `./app/config_<PIR name>.json`. If it does not, see [here](#create-config-file) to create a new one.
2. Depending on your CPU architecture you might want to specify a different one in `modules/psetggem/pset_ggm_c.go` line 4 for `amd64`. Right now it it set to `-march=ivybridge`.
3. Build the container with `make build`.
4. Run the container with `make <PIR scheme to benchmark>`. See the `Makefile` for all possible options.
5. Results of finished benchmark runs will be stored in `csv` format in `./app/<results file>.csv`. These results can be read while other benchmarks are still ongoing (do not change this file while the container is still running!).
6. Once the container has finished its work it will stop and will be removed. The full benchmarking results stay available at `./app/<results file>.csv`.
   The runtime is given in microseconds and the bandwidth is given in bytes.

To change the path of the input and output files, modify the `run` command in `Makefile` accordingly, where `path` specifies the input file path and `out` specifies the output file path.

> NOTE: For debugging purposes, the container does not get deleted once finished. To remove it after a run use `podman rm <container name>`. See the Makefile for the container names. 
> To see log ouputs for the containerized benchmarking suite use `podman logs -f benchmark`.

## How to run tests for the `tapir` package

This repository contains test cases for its different functionalities.
To run a test use the `go test` command and specify the desired package to test.
Please note, that some package and parameter combinations, e.g. Pointproofs (`modules/vc/pointproofs.go`), have an increase runtime and thus require manually increasing the timeout with `-timeout=<time duration, e.g., 600s>`.

Example:
```sh
go test tapir/<path to the package to test> -timeout <time duration, e.g., 600s>
```

Note that tests for large databases and PointProofs might require higher timeouts.

## Create Config File

The benchmarking suite takes a `json` file as input and reads from it the specified benchmarking configurations. 
Multiple different benchmarks can be specified and they will be executed sequentially. The outputs of all benchmarking results will be written as one `csv`-delimited line to the same output file (specified when running the benchmarking suite).

Each benchmark configuration must include the following set of parameters: 
- `PirType`: Specifies the (A)PIR protocol to benchmark. Requires an `int` input that maps to a [PIR Type](#pir-types).
- `VcType`: Specifies the vector commitment protocol to use in the benchmark. Requires an `int` input that maps to a [VC Type](#vc-types).
- `Repetitions`: Number of repetitions for this benchmark (`int`)
- `DbSize`: Database size (`int`).
- `PartSize`: Size of the Database partitions. Set this to `-1` for all schemes that don't use partitioning (i.e., all but SinglePass and Tapir). The partition size needs to cleanly divide the database size!
- `RecSize`: Database record lenght (`int`) in bytes. Default = `16`

Example:

```json
{
    "Configs": [
        { 
            "PirType": 6,
            "PirType": 2,
            "Repetitions": 25,
            "DbSize": 16,
            "PartSize": 4,
            "RecSize": 16
        },
        ... 
        { 
            "PirType": 1,
            "Repetitions": 2,
            "DbSize": 16,
            "PartSize": -1,
            "RecSize": 16
        }
    ]
}
```

## Parsing Evaluation Results

1. Ensure `python` and `numpy, pandas`  are installed.
2. Go to the eval folder using `cd eval`.
3. Add all csv results for (A)PIR to one csv file `results.csv` in `eval/`. If evaluating updates collect the results in a seperate csv file `results-update.csv`.
4. make new folder for temp outputs and go there `mkdir apir && cd apir`.
5. parse result file `python ../parse-csv.py ../pets_results.csv`. This generates one file for each (A)PIR and record size combination. 
   - These files contain the results displayed in the evaluation graphs in our paper.
6. Run the `make-table.csv` script from the `eval` folder: `python make-table.py apir/ output`.
7. To parse the update results run  ``mkdir upd && cd upd & python ../parse-update-csv.py ../results-update.csv`. additional files are created, specifying the type of update (0: ADD, 1: EDIT, 2: BOTH) and the number of applied updates.


## Local Build

This repository contains a containerized build and run environment for our benchmarks. 

The following instructions allow local building and execution of this code, but might have additional requirements.
For more information on these, please see the `container/Containerfile` and the documentation of required tools.


- Clone and build the libOTe library
    ```sh
    git clone https://github.com/osu-crypto/libOTe.git && \
    cd libOTe && \
    git checkout a403ec37c6a32148648b7d8fd66dc35318d9f99d && \
    git submodule update --init --recursive 

    python3 build.py \
    -DENABLE_REGULAR_DPF=ON -DENABLE_PIC=ON -DLIBOTE_SHARED=ON \
    --install=build
    ```
- Update paths in `modules/osu-crypto/osu_dpf.go` header to point to the local libOTe build path
    This can be achieved using
    ```sh
    sed -i -e "s|\/usr\/local\/go\/scr\/tapir\/libOTe|$(pwd)|g" $PWD/../modules/osu_crypto/osu_dpf.go  
    ```
- Depending on your system architecture the according flags in that header need to be set.
- Install Go requirements from repository root
    ```sh   
    cd ..
    go get ./...
    ```
- Build benchmarking binary
    ```sh
    go build -o bench benchmark/full/benchmark.go
    ```
- Create config files for the benchmarks (see [Create Config File](#create-config-file))
- Run the benchmarks
    ```sh
    ./bench --path=<path to json config> --out=<path to result csv>
    ```



## Troubleshooting

We experienced problems when building the container on MacOS when including the `libOTe` library. These issues did not occur on the Ubuntu system we ran experiments on (see the paper for details). 
Excluding this library from the code and `container/Containerfile` allowed us to run the container and hence all except for the DPF-based APIR scheme on MacOS.

