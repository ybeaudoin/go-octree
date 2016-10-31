# octree demo

A demo involving all the package's functions where we
* pick 100,000 random points from the unit cube,
* have the user select a partitioning method,
* create the octree with a termination criterion of 50 points,
* save the result to the JSON file "demo.json",
* read the octree back from the above file,
* plot a histogram and
* query the octree with a 1,000 points randomly chosen from the data set that went into its creation.

```go
package main

import(
    "bufio"
    "fmt"
    "github.com/ybeaudoin/go-octree"
    "math/rand"
    "log"
    "os"
    "strconv"
    "strings"
    "time"
)

func main() {
    const(
        numPts     = 100000
        numQueries = 1000
        terminal_N = 50
        /*Plot parameters*/
        plotWidth  = 500 //pixels
        plotHeight = 500 //pixels
        pngFile    = "demo.png"
    )

    //Pick random points from the unit cube
    rand.Seed(time.Now().UnixNano())
    points := make(octree.DataSet, numPts)
    for ptNo := 1; ptNo <= numPts; ptNo++ {
        points["#" + strconv.Itoa(ptNo)] = octree.DataCoords{rand.Float64(), rand.Float64(), rand.Float64()}
    }
    fmt.Print("\n- data created. ")
    pause()

    //Select a partitioning method
    option    := 0
    stdin     := bufio.NewReader(os.Stdin)
    methods   := []string{"Centroid", "DataMidPoint", "Geometric Median", "XYZ Medians"}
    fmt.Println("\n>>> Enter a number to select the corresponding partitioning method <<<")
    for k,v := range methods {
        fmt.Printf("(%d) %s", k+1, v)
        if k < len(methods) - 1 { fmt.Print(" | ") } else { fmt.Print(" : ") }
    }
    input, err1  := stdin.ReadString('\n')
    option, err2 := strconv.Atoi(strings.Trim(input, " \r\n"))
    for err1 != nil || err2 != nil || ! (option > 0 && option <= len(methods)) {
        fmt.Println("\aTry again.")
        input, err1  = stdin.ReadString('\n')
        option, err2 = strconv.Atoi(strings.Trim(input, " \r\n"))
    }

    //Create the octree with the selected method
    octree.Make(methods[option-1], terminal_N, &points)
    fmt.Print("\n- octree created. ")
    pause()

    //Display summary
    octree.Summarize()
    //octree.Summarize("demo.txt")
    fmt.Print("\n- octree summarized. ")
    pause()

    //JSON Export
    octree.Export("demo.json", false)
    fmt.Print("\n- octree exported. ")
    pause()

    //JSON Import
    octree.Import("demo.json")
    fmt.Print("\n- octree imported. ")
    pause()

    //Plot histogram
    octree.Histogram(plotWidth, plotHeight, pngFile)
    fmt.Print("\n- histogram created. ")
    pause()

    //Query the octree with points that went into its creation
    queries: for queryNo := 1; queryNo <= numQueries; queryNo++ {
        k := "#" + strconv.Itoa(rand.Intn(numPts) + 1)
        v := points[k]
        for _, identifier := range strings.Split(octree.Query(&v), ",") {
            if k == identifier {
                fmt.Printf("Query Key = %s, Point = %v -> PASS\n", k, v)
                continue queries
            }
        }
        fmt.Printf("Query Key = %s, Point = %v -> FAIL\n", k, v)
        log.Fatalln("\aQuery returned " + octree.Query(&v))
    }
    fmt.Println("\nSUCCESS!")
}
func pause() {
    fmt.Print("Press 'Enter' to continue...")
    bufio.NewReader(os.Stdin).ReadBytes('\n')
    return
}
```















