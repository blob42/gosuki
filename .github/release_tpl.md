{{title}}

{{summary}}


## What Changed

{{changelog}}

## Installation

### Arch Linux

```sh
paru install gosuki-git
```

You can build from the `PKGBUILD` file in `packages/arch/PKGBUILD`

### Debian

Full instructions at [https://git.blob42.xyz/gosuki.net/-/packages/debian/gosuki](https://git.blob42.xyz/gosuki.net/-/packages/debian/gosuki)

🖥️Setup this registry from the command line:

```sh
sudo curl https://git.blob42.xyz/api/packages/gosuki.net/debian/repository.key -o /etc/apt/keyrings/gosuki.asc

echo "deb [signed-by=/etc/apt/keyrings/gosuki.asc] https://git.blob42.xyz/api/packages/gosuki.net/debian trixie main" | sudo tee -a /etc/apt/sources.list.d/gosuki.list
sudo apt update
```

🖥️To install the package, run the following command:
```sh
sudo apt install gosuki={{deb-version}}
```

### From Source

**Gosuki Daemon:**

```sh
go install -tags systray github.com/blob42/gosuki/cmd/gosuki@latest
```

*note*: skip the tags flag if you don't need the feature

**Suki**:

```sh
go install github.com/blob42/gosuki/cmd/suki@latest
```


## New Contributors

{{contributors}}
