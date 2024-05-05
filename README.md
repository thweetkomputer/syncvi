# comp90020 assignmemt

Syncvi is a distributed terminal editor using Raft implemented by Golang.

## How to use

Run `make` to build it.

Run `./syncvi -h` to see how to use it.

## Structure

- `cmd` package defines the command line interface.
- `raft` package implements the Raft algorithm and the RPC server in raft layer and defines the storage interface.
- `editor` package implements the terminal editor and the RPC server in application layer and contains some tools like diff.
- `storage` package implements the storage interface of raft.

