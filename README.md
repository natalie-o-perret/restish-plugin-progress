# restish-plugin-progress

[![CI](https://github.com/natalie-o-perret/restish-plugin-progress/actions/workflows/ci.yml/badge.svg)](https://github.com/natalie-o-perret/restish-plugin-progress/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/natalie-o-perret/restish-plugin-progress.svg)](https://pkg.go.dev/github.com/natalie-o-perret/restish-plugin-progress)
[![License](https://img.shields.io/github/license/natalie-o-perret/restish-plugin-progress)](LICENSE)

`restish-plugin-progress` adds a streaming `progress` output formatter to
[Restish](https://rest.sh/).

SSE and NDJSON define how records are transported, but neither defines a
standard progress payload. This plugin uses the following small JSON contract
for each input item:

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
66% ████████████████░░░░░░░░  Deploy instances  2/3 steps  running: applying changes
```

The visual bar is rendered by
[`schollz/progressbar`](https://github.com/schollz/progressbar).

## Build And Install

```sh
go build -o restish-progress .
restish plugin install ./restish-progress --yes
```

## Use

Select the formatter with `-o progress`. For an SSE or NDJSON endpoint whose
events already use the plugin's JSON contract:

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

## Customise

The defaults use a cyan 24-character Unicode bar. Environment variables can
change the style without changing event payloads:

| Variable | Default | Purpose |
| --- | --- | --- |
| `RSH_PROGRESS_WIDTH` | `24` | Bar width from 1 to 200 |
| `RSH_PROGRESS_COLOR` | `cyan` | Filled-bar colour |
| `RSH_PROGRESS_FILL` | `█` | Filled character |
| `RSH_PROGRESS_HEAD` | `█` | Leading character |
| `RSH_PROGRESS_EMPTY` | `░` | Empty character |
| `RSH_PROGRESS_START` | empty | Bar prefix |
| `RSH_PROGRESS_END` | empty | Bar suffix |

Colours may be `black`, `blue`, `cyan`, `green`, `magenta`, `red`, `white`,
or `yellow`. Restish disables colours automatically for redirected output.

```sh
RSH_PROGRESS_WIDTH=32 \
RSH_PROGRESS_COLOR=magenta \
RSH_PROGRESS_FILL='━' \
RSH_PROGRESS_HEAD='╺' \
RSH_PROGRESS_EMPTY='─' \
restish example events --rsh-print b -o progress
```
