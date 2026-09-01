# restish-plugin-progress

[![CI](https://github.com/natalie-o-perret/restish-plugin-progress/actions/workflows/ci.yml/badge.svg)](https://github.com/natalie-o-perret/restish-plugin-progress/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/natalie-o-perret/restish-plugin-progress.svg)](https://pkg.go.dev/github.com/natalie-o-perret/restish-plugin-progress)
[![License](https://img.shields.io/github/license/natalie-o-perret/restish-plugin-progress)](LICENSE)

`restish-plugin-progress` adds a streaming `progress` output formatter to
[Restish](https://rest.sh/).

Each input item is an object with this shape:

```json
{
  "id": "deploy",
  "label": "Deploy instances",
  "state": "running",
  "current": 3,
  "total": 5,
  "message": "starting instance 4"
}
```

`label` falls back to `id`. `state` defaults to `running`. `current` and
`total` are optional, but must be supplied together.

The formatter redraws one terminal line when Restish enables terminal
formatting. Redirected or colour-disabled output uses one line per changed
record, which remains readable in logs and pipes.

```text
Deploy instances [#############-------] 66% (2/3 steps) running: applying changes
```

## Build And Install

```sh
go build -o restish-progress .
restish plugin install ./restish-progress --yes
```

## Use

Select the formatter with `-o progress`. For an SSE or NDJSON endpoint whose
events already use the progress record shape:

```sh
restish example events --rsh-print b -o progress
```

Use a Restish filter to map another event shape into the contract before
formatting it:

```sh
restish example events \
  -f 'body.data | {id, label, state, current, total, message}' \
  --rsh-print b \
  -o progress
```
