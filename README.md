# flatgeobuf

Native Go library implementing [FlatGeobuf](https://flatgeobuf.org/), a
performant binary encoding for geographic data based on
[FlatBuffers](https://flatbuffers.dev/).

## Project Status

Advancing rapidly toward alpha status. An alpha cut is expected by
August 31, 2023 and a `v1.0.0` by mid-November 2023 at the latest.

You can already peruse the documentation at the official Go docs
website, [here](https://pkg.go.dev/github.com/gogama/flatgeobuf).

## Getting Started

coming soon.

## Compatibility

coming soon.

## Package Map

coming soon.

## License

This project is licensed under the terms of the MIT License.

Some `*.fgb` files in `testdata/` are copied from the official
FlatGeobuf repository and licensed separately under the BSD-2-Clause
License. See `testdata/flatgeobuf/LICENSE`.

The code in package `flat` is generated using `flatc` from the official
GitHub repository's [FlatBuffer schema](https://github.com/flatgeobuf/flatgeobuf/tree/master/src/fbs).

## Acknowledgements

Thanks to @bjornharrtell for developing the FlatGeobuf specification and
@thehoneymad for getting me interested in it. Thanks to JetBrains, for
generously donating an open source license for their lovely GoLand IDE.

## Shameless Plugs

Geospatially, check out the [Overture](https://overturemaps.org/)
project.

Within Gogama projects, [Incite](https://github.com/gogama/incite) is a
fantastic library to smooth out working with AWS CloudWatch Logs, and
[httpx](https://github.com/gogama/httpx) is an excellent, if criminally
underused, robust HTTP client.
