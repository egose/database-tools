# Troubleshooting Guide

## Issue: "GOROOT is not set when Go is installed via asdf"

### Solution

If you're experiencing issues with GOROOT not being set after installing Go via asdf, you can follow these steps to address the problem:

1. Open your `~/.bashrc` file in a text editor:

   ```sh
   nano ~/.bashrc
   ```

2. Add the following bash function at the end of the file:

   ```sh
   set_asdf_go_root() {
     asdf current golang 2>&1 > /dev/null
     if [[ "$?" -eq 0 ]]
     then
       export GOBASE=$(asdf where golang)
       export GOROOT="$GOBASE/go"
     fi
   }

   set_asdf_go_root
   ```

   Save and exit the text editor.

3. Refresh your bash environment by either restarting your terminal or running:
   ```sh
   source ~/.bashrc
   ```

## Issue: "GOPROXY list is not the empty string, but contains no entries"

### Solution

1. Verify the existence of a valid `GOROOT` path by running the command:
   ```sh
   go env GOROOT
   ```
2. If `GOROOT` is not set, create a default `go.env` file under `GOROOT` containing the necessary configuration, including `GOPROXY`. Use the following commands:

   ```sh
   cat <<EOL | tee -a "$(go env GOROOT)/go.env"
   # This file contains the initial defaults for go command configuration.
   # Values set by 'go env -w' and written to the user's go/env file override these.
   # The environment overrides everything else.

   # Use the Go module mirror and checksum database by default.
   # See https://proxy.golang.org for details.
   GOPROXY=https://proxy.golang.org,direct
   GOSUMDB=sum.golang.org

   # Automatically download newer toolchains as directed by go.mod files.
   # See https://go.dev/doc/toolchain for details.
   GOTOOLCHAIN=auto
   EOL
   ```
