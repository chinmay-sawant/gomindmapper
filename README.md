# GoMindMapper `cmd` analyzer

This folder contains a small Go-based code analyzer used to scan a repository and produce two JSON artifacts:

- `functions.json` — full list of discovered functions and the raw calls found inside each function body (before user-call filtering). This preserves the original `Calls` arrays for debugging.
- `functionmap.json` — cleaned function relationship map containing only user-defined call edges. Entries that do not call any user-defined functions are omitted.
- `removed_calls.json` — report listing calls removed from `functions.json` when filtering out non-user calls. Useful for tuning exclusion/whitelisting rules.

Files
- `main.go` — orchestrates scanning the repository, extracting functions, and building the function map.
- `analyzer/utils.go` — contains `FindFunctionBody`, `FindCalls`, and `GetModule`.
- `analyzer/fileops.go` — writes `functions.json` and `removed_calls.json`.
- `analyzer/types.go` — data structures used by the analyzer.

Usage

From the repository root run (Windows `cmd.exe` / PowerShell):

```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper"
go run cmd/main.go
```

Optional: analyze a specific subpath

```cmd
go run cmd/main.go -path ./EmployeeApp
```

Outputs
- `functions.json` — full details of discovered functions (kept as-is for debugging and further processing).
- `functionmap.json` — simplified graph of user-defined function call relations (only functions that call other user functions are included).
- `removed_calls.json` — report of which calls were filtered out when producing `functions.json`.

Customization
- To change which low-level or framework calls are excluded, edit `cmd/analyzer/utils.go`:
  - The `FindCalls` function contains regex/filter rules for excluding regex helpers, error handling, wait-group methods, and standard library packages.
- To change how non-user calls are identified, edit `cmd/analyzer/fileops.go` where user package prefixes are computed and used to filter calls. You can implement a whitelist for framework prefixes (e.g., `c` for Gin context) by modifying the `userPrefixes` logic or by adding a `whitelist` map.

Notes
- `functionmap.json` intentionally omits functions that don't call other user-defined functions to keep the graph focused on actual function-to-function relationships.
- `functions.json` still includes all discovered functions and the raw set of calls (useful for debugging and tuning filters).

Next steps (suggested)
- Add a CLI flag to control whether `functionmap.json` should omit functions with no calls.
- Use `go list` to map module/package paths for more robust user-call detection instead of the simple prefix heuristic.
- Export the function map in CSV or GraphML for import into visualization tools.

If you'd like, I can add a `-whitelist` flag to keep certain prefixes (for example `c` or `router`) in the `functionmap.json` output.
