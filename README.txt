Authors: Chen Zhao, Lezhou Lang

SyncVi
======

SyncVi is a distributed, collaborative terminal text editor implementing the Raft consensus algorithm, written in Go.

Installation and Building
=========================

Run `make` to build it.

Run `go install` to install it.

The following command can be used to collaboratively edit a file with 3 members.

(Every one first runs `syncvi clean` to clean the data.)

For the first member:

    syncvi start t --peers=127.0.0.1:8888,127.0.0.1:8889,127.0.0.1:8890 --nodes=127.0.0.1:18888,127.0.0.1:18889,127.0.0.1:18890 --me=0

For the second member:

    syncvi start t --peers=127.0.0.1:8888,127.0.0.1:8889,127.0.0.1:8890 --nodes=127.0.0.1:18888,127.0.0.1:18889,127.0.0.1:18890 --me=1

For the third member:

    syncvi start t --peers=127.0.0.1:8888,127.0.0.1:8889,127.0.0.1:8890 --nodes=127.0.0.1:18888,127.0.0.1:18889,127.0.0.1:18890 --me=2

Run `syncvi -h` to see how to use it in detail.

Details of Code Structure
=========================

- `cmd` package defines the command line interface.
- `raft` package implements the Raft algorithm and the RPC server in raft layer and defines the storage interface.
- `editor` package implements the terminal editor and the RPC server in application layer and contains some tools like diff.
- `storage` package implements the storage interface of raft.

