# 14-nested-gspkg

Demonstrates nested single-file package dependencies.
The `tools` package also uses an absolute-style import alias:
`@/* = "src/*.gs"`.

```text
app.gs
  -> vendor/tools.gspkg
       -> vendor/helper.gspkg
```

Run:

```powershell
gs run
```

Rebuild package files:

```powershell
gs pack packages/helper-src packages/tools-src/vendor/helper.gspkg
gs pack packages/tools-src vendor/tools.gspkg
```

Bundle:

```powershell
gs bundle app.gs dist/app.bundle.gs
```
