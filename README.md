# gdylib

Module providing adding following load commands:
* `LC_LOAD_DYLIB`
* `LC_LOAD_WEAK_DYLIB`
* `LC_RPATH`

This module is heavily inspired/guided by [insert_dylib](https://github.com/tyilo/insert_dylib), [install_name_tool](https://www.unix.com/man-page/osx/1/install_name_tool/) projects.

# Usage

```golang
cfg := gdylib.Config{
  Remove: true, // remove signature
  Type: gdylib.RPATH,
  Binary: "/some/path/to/the/binary",
  Reader: r, // you can pass reader or path to the binary
}

r, err := gdylib.Patch(&cfg)

f, err := os.Create("output")
if err != nil {
  panic(err)
}
defer f.Close()

io.Copy(f, r)
}
