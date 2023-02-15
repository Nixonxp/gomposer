# Gomposer
## Tool for download private vendor packages with access tokens

## Installation

Once you have [installed Go][golang-install], install the `gomposer` tool.

**Note**: If you have not done so already be sure to add `$GOPATH/bin` to your
`PATH`.

To get the latest released version use:

### Go version < 1.16

```bash
GO111MODULE=on go get github.com/Nixonxp/gomposer@v0.0.2
```

### Go 1.16+

```bash
go install github.com/Nixonxp/gomposer@v0.0.2
```

## Description

For work required 2 files `go.private`, `go.mod` in root folder

### `go.private` need for private packages url list with main branch for this repo
Example:
```txt
https://<access-token>@<repo-url> <branch>
```

Module created <b>vendor-private</b> directory, where cloned repos

In `go.mod` necessary to indicate the replacement of the target section of the module load on yours
and required repo name in modules list
```mod
replace <repo-name> => ./vendor-private/<repo name>
```