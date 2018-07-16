# Contributing

Thanks for taking the time to join the community and helping out! These
guidelines will help you get started with the Heighliner project.

Please note that we have a [CLA sign off](https://cla-assistant.io/manifoldco/heighliner).

## Building from source

### Prerequisites

1. Install 1Go1

    Heighliner requires [Go 1.9][1] or later.

2. Install `dep`

    Heighliner uses [dep][1] for dependency management.

    ```
    go get -u github.com/golang/dep/cmd/dep
    ```

### Downloading the source

To reduce the size of the repository, Heighliner does not include a copy of its
dependencies. It uses [dep][2] to manage its dependencies.

We might change this in the future, but for now, you can use the following
commands to fetch the Heighliner source and its dependencies:

```
go get -d github.com/manifoldco/heighliner
cd $GOPATH/src/github.com/manifoldco/heighliner
make vendor
```

Go has strict rules when it comes to the location of the source code in your
`$GOPATH`. The easiest way to develop is to rename the Heighliner git remote
location and substitute your own fork for `origin`. We want to ensure that the
repository remains at `$GOPATH/src/github.com/manifoldco/heighliner` on disk.

```
git remote rename origin upstream
git remote add origin git@github.com:jelmersnoeck/heighliner.git
```

### Building

To build the binaries, run:

```
make bins
```

This will put all the binaries into the `./bins` folder. These binaries are
compiled for your local machine.

To compile a docker image to deploy in your local cluster, there are two
options. The first options is to run

```
make docker-dev
```

This will generate the binary on the host - your machine - and put it in a
Docker image.

To create a more official image, you can run:

```
make docker
```

This will install all dependencies in the Docker image and build the container
in that image. This means that all build artifacts are linked within the same
Docker structure.

### Testing

Once you have Heighliner built, you can run the tests:

```
make test
```

We also have a set of linters that we require, these can be run as follows:

```
make lint
```

[1]: https://golang.org
[2]: https://github.com/golang/dep
