# mok

`mok` is a simple tool to serve JSON files locally.

## why

when building applications that consume JSON (especially frontends calling APIs), you might want to:
- avoid rate limiting during development
- work offline or reduce network dependencies
- iterate quickly without waiting for remote requests
- be a good netizen and minimize unnecessary traffic

`mok` lets you serve local JSON files or download remote data once and serve it locally, giving you fast, reliable access to your test data.

## install

install `mok` with:

```console
go install github.com/rcastellotti/mok@latest
````

## usage

```console
$ go run mok.go testdata/*.json https://api.github.com/repos/rcastellotti/mok
```

### passsing direct input via `-s`
```console
$ go run mok.go  -s '{"num":3.14,"fav":["b","e","a","r"]}'
```

### passsing direct input via stdin

```console
$ echo '{"num":3.14,"fav":["b","e","a","r"]}' | go run mok.go
```

`mok` renders an index of all served files at the root path `/`.
the endpoint reads the `Accept` header to determine the response format, an example:

```console
$ curl -s -H "Accept: application/json" http://localhost:9172/ | jq
[
  {
    "FilePath": "testdata/a.json",
    "URLPath": "/a.json"
  },
  {
    "FilePath": "testdata/b.json",
    "URLPath": "/b.json"
  },
  {
    "FilePath": "testdata/github-repo-info.json",
    "URLPath": "/github-repo-info.json"
  },
  {
    "FilePath": "/var/folders/ym/11b11s_s0pd8gyncvvctf9c80000gp/T/mok-jsonplaceholder.typicode.com.2676125774.json",
    "URLPath": "/mok-jsonplaceholder.typicode.com.2676125774.json"
  }
]
```

for more information: `mok -h`
