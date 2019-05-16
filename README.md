# DapperDox&nbsp;&nbsp;&nbsp;[![Build Status](https://travis-ci.org/frinka/dapperdox.svg?branch=master)](https://travis-ci.org/frinka/dapperdox) [![Download](https://img.shields.io/badge/release-v1.2.3green.svg) ](https://github.com/DapperDox/dapperdox/releases/tag/v1.1.1)

### Beautiful, integrated, OpenAPI documentation.

> Themed documentation generator, server and API explorer for OpenAPI (Swagger) Specifications. Helps you build integrated, browsable reference documentation and guides. For example, the [Companies House Developer Hub](https://developer.companieshouse.gov.uk/api/docs/) built with the alpha version.

![DapperDox logo](/docs/logo-dapperdox-github.png "DapperDox")

## Features

* Author full documentation in GitHub Flavoured Markdown.
* Document multiple API specifications as a suite of cross-referenced products.
* Seamlessly overlay content onto the automatically generated reference documentation.
* Integrate the built-in API explorer with your APIs and authentication model.
* Proxy your developer platform, allowing full integration of API key management.
* Choose from multiple themes, or create your own.

## Documentation and usage

Detailed usage instructions are available on the [DapperDox website http://dapperdox.io](http://dapperdox.io).

## Quickstart

We provide the lastest release builds for the most common operating systems and architectures, which you
can download from [releases](https://github.com/DapperDox/dapperdox/releases),
or you can build from source by cloning this repo..
See [downloads](http://dapperdox.io/download/downloads) for further information.

### Build from source

First build DapperDox (this assumes that you have your golang environment configured correctly):
```bash
go get && go build
```

### Running DapperDox

Start up DapperDox, pointing it to your OpenAPI 2.0 specification file:

```
./dapperdox -spec-dir=<location of OpenAPI 2.0 spec>
```

DapperDox looks for the file `swagger.json` at the `-spec-dir` location, and builds reference documentation for the OpenAPI specification it finds. For example, the obligatory *petstore* OpenAPI specification is provided in the `examples/specifications/petstore` directory, so
passing parameter `-spec-dir=examples/specifications/petstore` will build the petstore documentation.

DapperDox will default to serving documentation from port 3123 on all interfaces, so you can point your 
web browser at http://127.0.0.1:3123 or http://localhost:3123.

For an out-of-the-box example, execute the example run script:

```bash
./run_example.sh
```

This demonstrates many of the configuration options available. See [configuration](http://dapperdox.io/docs/configuration-guide).

## Acknowledgements

Many thanks to [Ian Kent](https://github.com/ian-kent) who spiked the Golang implementation of DapperDox
as part of a bigger project. His commit history was lost when DapperDox was extracted into a stand-alone
project, but its core retains his valuable work.

[David Mort](https://github.com/davidmort) for painstaking testing, and bug fixing.





