# goblin-starter

A reference WASM goblin for the [ai-goblins](https://github.com/ai-goblins) platform.
Fork this repo and replace the business logic with your own.

---

## What it does

Once per day, at a random time within a configurable window, this goblin sends a
short personalised salutation to Claude's prompt. Claude renders it into a greeting
and delivers it through whichever channels the goblin clone is configured to use.

If the goblin runs before the scheduled time, it skips silently. If it runs after
the window has already fired today, it also skips. State is managed automatically.

---

## Blueprint arguments

| Argument | Type | Default | Description |
|---|---|---|---|
| `name` | string | `"friend"` | Recipient's name used in the greeting |
| `earliest_hour` | integer | `8` | Earliest UTC hour the salutation may be sent (inclusive) |
| `latest_hour` | integer | `20` | Latest UTC hour the salutation may be sent (exclusive) |

## Output data

When the goblin fires it writes the following into `output.data`, which becomes
`{wasm_data}` in your blueprint's prompt:

```json
{
  "name":        "Alice",
  "time_of_day": "morning"
}
```

`time_of_day` is one of `morning` (00:00–11:59), `afternoon` (12:00–16:59), or
`evening` (17:00–23:59) in UTC.

### Example prompt

```
Say hello to {name} with a warm {time_of_day} greeting.
Keep it to one sentence.

Data: {wasm_data}
```

---

## Project layout

```
goblin-starter/
  goblin.go          ← business logic (testable as plain Go)
  main.go            ← WASM entry point (thin SDK wrapper)
  goblin_test.go     ← unit tests
  testdata/
    input.json       ← sample input for local runs
  go.mod
  go.sum
```

The split between `main.go` and `goblin.go` is deliberate: all business logic
lives in `goblin.go` and is unit-tested with `go test` — no WASM toolchain needed
for testing.

---

## Building and running locally

Requires Go 1.21+ (native WASI support — no TinyGo needed).

### Run tests

```bash
go test ./...
```

### Compile to WASM

```bash
GOOS=wasip1 GOARCH=wasm go build -o goblin-starter.wasm .
```

### Run locally (using the platform's dev tooling)

```bash
# from the backend repo root
misc/wasm-run.sh goblin-starter testdata/input.json
```

---

## Key patterns

### Parsing arguments and state

`input.Arguments` and `input.State` arrive as `map[string]any` — Go's generic
representation of a decoded JSON object. To work with them in a type-safe way,
define structs with `json:` tags and use a JSON round-trip:

```go
type goblinArgs struct {
    Name string `json:"name"`
}

func parseArgs(raw map[string]any) (goblinArgs, error) {
    a := goblinArgs{Name: "friend"} // set defaults first
    data, _ := json.Marshal(raw)
    if err := json.Unmarshal(data, &a); err != nil {
        return goblinArgs{}, fmt.Errorf("unmarshal args: %w", err)
    }
    return a, nil
}
```

Marshal puts the map back into JSON bytes; unmarshal converts those bytes into
the typed struct using the struct tags. Nested types — slices, nested structs —
are handled for free. See `goblin.go` for the full implementation, and the
[Developer Guide](https://github.com/ai-goblins/goblin-sdk/blob/main/DEVELOPER_GUIDE.md#64-parsing-arguments-and-state)
for a detailed explanation.

---

## Forking guide

1. Fork this repo and rename it (`my-goblin`, `weather-check`, etc.).
2. Update `module` in `go.mod` to match your new repo path.
3. Replace `goblinArgs`, `goblinState`, and `run()` in `goblin.go` with your logic.
4. Use `sdk.HTTPGet(url)` to fetch external data (domains must be allowlisted in the blueprint).
5. Update `goblin_test.go` to cover your new behaviour.
6. Compile and deploy using the platform tooling.

See the [Goblin Developer Guide](https://github.com/ai-goblins/goblin-sdk/blob/main/DEVELOPER_GUIDE.md)
for the full I/O contract, state management rules, host function reference, and security constraints.

---

## License

MIT
