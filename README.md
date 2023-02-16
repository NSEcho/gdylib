# gdylib

Module providing adding following load commands:
* `LC_LOAD_DYLIB`
* `LC_LOAD_WEAK_DYLIB`
* `LC_RPATH` (TODO)

This module is heavily inspired/guided by [insert_dylib](https://github.com/tyilo/insert_dylib), [install_name_tool](https://www.unix.com/man-page/osx/1/install_name_tool/) projects.

# Usage

```golang
package main

import (
	"github.com/lateralusd/gdylib"
	"io"
	"os"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r, err := gdylib.Run(os.Args[1], os.Args[2],
		gdylib.WithLoadType(gdylib.DYLIB))
	if err != nil {
		panic(err)
	}

	nf, err := os.Create(os.Args[3])
	if err != nil {
		panic(err)
	}
	defer nf.Close()

	io.Copy(nf, r)
}
```

```bash
$ go run main.go a.out @executable_path/FridaGadget.dylib new_file
$ otool -l new_file | tail
  cmdsize 16
  dataoff 32920
 datasize 0
Load command 16
          cmd LC_LOAD_DYLIB
      cmdsize 64
         name @executable_path/FridaGadget.dylib (offset 24)
   time stamp 0 Thu Jan  1 01:00:00 1970
      current version 0.0.0
compatibility version 0.0.0
```