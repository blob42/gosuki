## Development notes

The entrypoints for CLIs are:

```sh
cmd/gosuki/main.go
cmd/suki/suki.go
```

So to run via Go (and debug it)

```sh
go run ./cmd/gosuki
```

or

```sh
go run ./cmd/gosuki
```

## VSCodium/VSCode setup for running / debugging:

```sh
.vscode/launch.json
```

```json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug gosuki import pocket",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceRoot}/cmd/gosuki",
            "showLog": true,
            // "debugAdapter": "dlv-dap",
            "args": ["import", "pocket", "--debug=trace", "${workspaceRoot}/debug.csv"]
        }
    ]
}
```

For MacOS:

```sh
.vscode/settings.json
```

```json
{
  "go.goroot": "/opt/homebrew/Cellar/go/1.24.3/libexec"
}
```

Extensions:

- https://open-vsx.org/vscode/item?itemName=golang.Go
