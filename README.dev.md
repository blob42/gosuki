## Development notes

The entry points for the CLI are:

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
go run ./cmd/suki
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

### For MacOS:

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

## Debugging

### Attach to running process with dlv

- This example uses `dlv` to launch gosuki in server mode with TUI using a test config file and database and enabling debug log for Chrome based browsers.

```sh
dlv debug --headless --listen 127.0.0.1:38697 ./cmd/gosuki/ -- -c ~/.config/gosuki/config.test.toml --db=/tmp/gosuki.db --debug=none,chrome=debug start
```

- Then attach to the dlv session using your IDE DAP plugin after setting a breakpoint
