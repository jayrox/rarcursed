package main

import (
	"bytes"
	"flag"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
    "math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	flagdebug   = flag.Bool("d", true, "show debug output")
	flagdir     = flag.String("dir", "cwd", "directory to scan. default is current working directory (cwd)")
	flagefr     = flag.String("rem", "", "extra files to remove (i.e. *.nfo, *.sfv), default does not remove extras")
	flagminsize = flag.Int64("min", 200000000, "minimum file size to include in scan. default is 200MB") // 3MB
	flagtest    = flag.Bool("test", false, "tests process, doesn't delete files.")
	rf          rcdFlags
	extractext  = []string{".rar", ".zip", ".7z"}
	whitelist   = []string{
		".mkv", ".mp4", ".avi",
		".bup", ".ifo", ".vob",
	}
)

func main() {
	flag.Parse()
	// Print the logo :P
	printLogo()

	// Root folder to scan
	fpAbs, _ := filepath.Abs(flagString(flagdir))
	rf.Dir = fpAbs

	rf.Min = flagInt(flagminsize)

	if flagString(flagdir) == "cwd" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		rf.Dir = dir
	}
	if flagBool(flagtest) {
		fmt.Println("**Test mode enabled!")
	}
	fmt.Printf("Scanning directory: %s\n", rf.Dir)
	fmt.Println("_____________________")

	ok := test7zip()
	if !ok {
		fmt.Println("7zip has not been found, verify 7-Zip is installed and in the PATH env variable.")
		return
	}
	fmt.Println("_____________________")

	i := folderWalk(rf.Dir)
	if i < 1 {
		fmt.Println("No archives found.")
	}
}

func flagString(fs *string) string {
	return fmt.Sprint(*fs)
}

func flagInt(fi *int64) int64 {
	return int64(*fi)
}

func flagBool(fb *bool) bool {
	return bool(*fb)
}

func folderWalk(path string) (i int64) {
	i = 0
	var err = filepath.Walk(path, func(path string, _ os.FileInfo, _ error) error {
		for _, x := range extractext {
			if filepath.Ext(path) == x && !rarPartXX(path) {
				var ok bool

				ok = testcrc32(path)
				if ok == false {
					fmt.Println("Test Failed - skipping archive")
					fmt.Println("_________")
					continue
				}

				ok = testarchive(path)
				if ok == false {
					fmt.Println("Test Failed - skipping archive")
					fmt.Println("_________")
					continue
				}

				ok = extract(path, "./test/e/")

				if ok == false {
					printDebug("Extract failed %s\n", "")
				} else {
					i = i + 1
					cleanPath(path)
				}
				fmt.Println("_________")
			}
		}
		return nil
	})
	if err != nil {
		printDebug("er: %+v\n", err)
	}
	return
}

func test7zip() (ok bool) {
	ok = true
	printDebug("Testing: 7-Zip%s\n", "")
	out, err := exec.Command("7z").Output()
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Contains(out, []byte("7-Zip")) {
		fmt.Println("7-Zip not found.")
		ok = false
	}

	if ok {
		fmt.Println("7-Zip found")
	}
	return
}

func testarchive(path string) (ok bool) {
	ok = true
	printDebug("Testing: %s\n", path)
	out, err := exec.Command("7z", "t", path).Output()
	if err != nil {
		if strings.Contains(err.Error(), "CRC") {
			fmt.Println("err: ", err)
			ok = false
		}
	}
	if bytes.Contains(out, []byte("CRC")) {
		fmt.Println("ERROR: CRC Failed")
        fmt.Println(path)
		ok = false
	}

	if ok {
		fmt.Println("Archive Test PASSED")
	}
	return
}

func testcrc32(path string) (ok bool) {
	ok = true
	d := filepath.Dir(path)
	var err = filepath.Walk(d, func(fpath string, f os.FileInfo, _ error) error {
		if filepath.Ext(fpath) == ".sfv" {
            printDebug("Found sfv\n%s\n", fpath)

			dat, err := ioutil.ReadFile(fpath)
			if err != nil {
				return err
			}
			temp := strings.Split(string(dat), "\n")
			for i, x := range temp {
                if math.Remainder(float64(i), 25) == 0 {
                    fmt.Println("")
                }
                if len(x) < 1 {
                    continue
                } 
                if string(x[0]) == ";" {
                    continue
                }
                
                result := strings.Fields(x)
				if len(result) > 0 {
					rcrc32 := result[len(result)-1]
					h, err := getHash(d + "\\" + result[0])
					if err != nil {
						//fmt.Println(err)
					}
					s := strconv.FormatInt(int64(h), 16)
					s = strings.TrimLeft(s, "0")
					rcrc32 = strings.TrimLeft(rcrc32, "0")
                    s = strings.ToUpper(s)
                    rcrc32 = strings.ToUpper(rcrc32)
					if s != rcrc32 {
						ok = false
						printDebug("CRC does not match:\n%s\n%s - %s\n", result[0], rcrc32, s)
					} else {
						printDebug("%s", ".")
					}
				}
			}
			printDebug("%s\n", "")
		}

		return nil
	})
	if err != nil {
		ok = false
		fmt.Println(err)
	}
	return
}

func getHash(filename string) (uint32, error) {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	h := crc32.NewIEEE()
	h.Write(bs)
	bs = nil
	return h.Sum32(), nil
}

func extract(path, destpath string) (ok bool) {
	d := filepath.Dir(path)
	ok = true
	printDebug("Extracting: %s\nDestination: %s\n", path, d)
	cmd := exec.Command("7z", "x", path, "-o"+d, "-y")
	if err := cmd.Run(); err != nil {
		// error code 2 means Fatal Error from 7zip executable
		// possible cause could be a missing file in a multipart archive
		if err.Error() == "exit status 2" {
			printDebug("Extration error: %+v\n", err)
			ok = false
		}
	}

	if !ok {
		printDebug("Extraction failed %s\n", "")
	} else {
		printDebug("Extraction complete %s\n", "")
	}
	return
}

func cleanPath(path string) {
	wd := filepath.Dir(path)
	printDebug("Cleaning: %s\n", wd)
	var err = filepath.Walk(wd, func(fpath string, f os.FileInfo, _ error) error {
		keepfile := false
		for _, x := range whitelist {
			// Check if file extension is in the keeper list
			if filepath.Ext(fpath) == x {
				keepfile = true
			}
			// Check if file size > min, prevent accidental deletion of wanted files
			if f.Size() > rf.Min {
				keepfile = true
			}
		}

		// Don't delete root folder
		if fpath == wd {
			keepfile = true
		}

		// Test mode to prevent file deletion
		if flagBool(flagtest) {
			keepfile = true
			printDebug("Testing - %s\n", "deletion skipped")
		}

		// Delete file if it doesnt match any of the keeper rules
		if keepfile == false {
			printDebug("Deleting: %s\n", fpath)
			err := os.Remove(fpath)
			if err != nil {
				printDebug("File deletion error: %+v\n", err)
			}
		}

		return nil
	})
	if err != nil {
		printDebug("Clean err: %+v\n", err)
	}
}

func rarPartXX(path string) bool {
	if !strings.HasSuffix(path, ".rar") {
		return false
	}
	if strings.HasSuffix(path, ".part01.rar") || strings.HasSuffix(path, ".part1.rar") {
		return false
	}
	if strings.HasSuffix(path, ".rar") && !strings.Contains(path, ".part") {
		return false
	}
	return true
}

// Check err
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Only print debug output if the debug flag is true
func printDebug(format string, vars ...interface{}) {
	if *flagdebug {
		if vars[0] == nil {
			fmt.Println(format)
			return
		}
		fmt.Printf(format, vars...)
	}
}

// Hold flag data
type rcdFlags struct {
	Dir                string
	Debug              bool
	ExtraFilesToRemove string
	Min                int64
}

// Print the logo, obviously
func printLogo() {
    fmt.Println("██████╗  ██████╗██████╗") 
    fmt.Println("██╔══██╗██╔════╝██╔══██╗")
    fmt.Println("██████╔╝██║     ██║  ██║")
    fmt.Println("██╔══██╗██║     ██║  ██║")
    fmt.Println("██║  ██║╚██████╗██████╔╝")
    fmt.Println("╚═╝  ╚═╝ ╚═════╝╚═════╝ rarcursed")
    fmt.Println("")
}
