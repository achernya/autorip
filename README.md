# autorip

autorip is a digital archival and preservation automation tool for
optical media. ["Ripping"](https://en.wikipedia.org/wiki/Ripping) a
modern Blu-ray or DVD is a fairly manual process, as it involves
identifying the content, determining which parts to extract, and
finally naming the output file(s) correctly.

autorip aims to automate the workflow around an existing tool that
performs the actual ripping operations.

## How it works

autorip itself does not contain any code to read or even interact with
optical media. Instead, it uses an underlying tool (makemkv, in this
case), which interacts with the disc drive.

autorip uses the metadata reported by this underlying tool to analyze
the media and determine what it is (e.g., a movie or a TV series) by
cross-referencing it with the data available in the [IMDb
non-commercial
datasets](https://developer.imdb.com/non-commercial-datasets/).  This
dataset is stored and indexed locally, enabling reproducible analysis
and avoiding the pitfalls that come with external APIs such as
rate-limiting or unexpected data changes.

## Building

autorip is written in [Go](https://go.dev/) and requires a full go
toolchain to build. autorip also depends on protobuf; make sure that
you have both [protobuf](https://protobuf.dev/downloads/) and
[protoc-gen-go](https://protobuf.dev/getting-started/gotutorial/#compiling-protocol-buffers)
installed and available in `PATH`. Then, run

```
make
```

## Using

1. make a copy of `example_config.yaml` and copy it over to
   `$HOME/.autorip.yaml`. Fill in the values there to suit your needs.
1. Run `autorip imdb index` to fetch IMDb data and build the local
   on-disk databases and search index. This should take about 2-3
   minutes once the download completes.
1. [Optional] Query the index with `autorip imdb search "search
   terms"`. The output will be in JSON. [jq](https://jqlang.org/) is a
   great companion for pretty-printing and filtering this data. The
   underlying search engine is [Bleve](https://blevesearch.com/) and
   the fully [query
   syntax](https://blevesearch.com/docs/Query-String-Query/) is
   supported. `AverageRating` and `NumVotes` are numerical columns
   available for filtering.
1. [Optional] Analyze a disc with `autorip analyze`
1. Preserve a disc with `autorip rip`

## Known Issues

autorip is currently alpha software. It is incomplete, and has at
least the following issues:

1. The heuristic for determining which content is "right" is mostly
   based on the runtime. As a result, extended editions cannot be
   correctly identified since their runtimes are too substantially
   different from what is in IMDb.
1. Discs that contain multiple versions of a piece of media (e.g.,
   theatrical and also extended) will currently always prefer the
   longer version, even if it results in being unable to identify the
   content.
1. Discs whose metadata contains no separator characters between words
   cannot be properly identified.
1. Support for TV series is currently unimplemented.

## Disclaimer

autorip should only be used for non-commercial, digital archival and
preservation purposes. You are obligated to follow all applicable laws
and regulations while using this software, including but not limited
to the software license of this program and its dependencies, as well
as the license of the [IMDb
datasets](https://developer.imdb.com/non-commercial-datasets/) it
leverages.

## Use of LLMs

This software was developed without the use of any LLMs, coding
assistants, or "smart" autocomplete.

## License

BSD 2-Clause License
