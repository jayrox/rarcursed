# rarcursed
recursive rar extractor and deleter  
-----
rarcursed uses 7zip to extract files, make sure it is available in your PATH environment.

--------

# Get Started
install Go and setup your [GOPATH](http://golang.org/doc/code.html#GOPATH)

# Clone the repository
`git clone https://github.com/jayrox/rarcursed`

# Build
```
cd rarcursed  
go build
```

# Run
```
rdc -dir /path/to/rars
```

# Command-line Variables
```
-d    - debug output
-dir  - directory to scan, extract, cleanup
-min  - minimum file size to keep
-test - run process but don't delete files
-rem  - extra file types to clean up (*.nfo, *.sfv)
```
