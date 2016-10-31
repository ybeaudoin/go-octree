/*===== Copyright 2016, Webpraxis Consulting Ltd. - ALL RIGHTS RESERVED - Email: webpraxis@gmail.com ===========================
 *  Package octree:
 *      import "octree"
 *  Overview:
 *      package for creating, reporting, importing, exporting and querying point-region octrees.
 *  Types:
 *      DataCoords
 *          Array for a data point's R^3 coordinates, i.e., [1.,2.,3.]
 *      DataSet
 *          Map for the R^3 data points keyed on identifiers, i.e., "Pt1":[1.,2.,3.], "Pt2":[4.,5.,6.], etc.
 *  Variables:
 *      MaxIterations
 *          Maxinum number of iterations for the geometric-median partition method
 *      Tol
 *          Convergence tolerance for the geometric-median partition method
 *  Functions:
 *      Export(file string, compact bool)
 *          Exports the octree and its meta data to a specified file using the JSON format with or without newlines
 *          and identations.
 *      Histogram(plotWidth, plotHeight int, pngFile string)
 *          Plots a histogram of the leaf point counts and saves it to the specified PNG file.
 *          The population mean (mu) and standard deviation (sigma) of the leaf counts are also illustrated.
 *      Import(file string)
 *          Imports an octree and its meta data from a specified JSON file.
 *      Make(method string, terminal_N int, refPoints *DataSet)
 *          Creates a point-region octree recursively using a specified partitioning method and termination criterion.
 *      Query(refQueryPt *DataCoords) string
 *          Traverses an octree top-down to get the data keys at the leaf node corresponding to the octant in
 *          which lies the specified query point.
 *      Summarize(output ...string)
 *          Outputs the meta data and various statistics regarding the octree to a specified file or Stdout.
 *  History:
 *      v1.0.0 - October 26, 2016 - Original release.
 *============================================================================================================================*/
package octree

import(
    "bitbucket.org/binet/go-gnuplot/pkg/gnuplot"
    "encoding/json"
    "fmt"
    "github.com/cznic/mathutil"
    "github.com/dustin/go-humanize"
    "io/ioutil"
    "log"
    "math"
    "os"
    "path/filepath"
    "runtime"
    "sort"
    "strings"
    "text/template"
    "time"
)
//Exported ---------------------------------------------------------------------------------------------------------------------
type(
    DataCoords    [3]float64            //array for a data point's R^3 coordinates
    DataSet       map[string]DataCoords //map for the R^3 data points keyed on identifiers
)
var(
    MaxIterations = 1000                //max no of iterations for the geometric-median method of partitioning
    Tol           = 1.0E-12             //convergence tolerance for the geometric-median method of partitioning
)

func Export(file string, compact bool) {
/*         Purpose : Exports the octree and its meta data to a specified file using the JSON format with or without newlines
 *                   and identations.
 *       Arguments : file    = filename for the JSON output,
 *                   compact = boolean flag for compact mode.
 *         Returns : None.
 * Externals -  In : jsonNode, jsonOctree, _octree, _stats
 * Externals - Out : None.
 *       Functions : halt
 *         Remarks : None.
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if _stats.SIZE == 0  { halt("there's no octree to export") }
    if file        == "" { halt("the filename was not specified") }

    var output []byte

    writer, err := os.Create(file) //open file for write
    if err != nil { halt("os.Create - " + err.Error()) }
    defer writer.Close()

    nodeData := make([]jsonNode, _stats.SIZE)
    for k, v := range _octree {
        if v.N > _stats.STOP { //parent node
            nodeData[k] = jsonNode{ ID:       k,
                                    N:        v.N,
                                    CENTER:   &(_octree[k].CENTER),
                                    CHILDREN: &(_octree[k].CHILDREN) }
        } else { //leaf node
            nodeData[k] = jsonNode{ ID:       k,
                                    N:        v.N,
                                    KEYS:     v.KEYS }
        }
    }
    jsonData := jsonOctree{ HOW:    _stats.HOW,
                            STOP:   _stats.STOP,
                            TIME:   _stats.TIME,
                            SIZE:   _stats.SIZE,
                            OCTREE: nodeData }
    if compact { output, err = json.Marshal(jsonData) // create the JSON output
    } else     { output, err = json.MarshalIndent(jsonData, "", " ") }
    if err != nil { halt("Marshal/MarshalIndent - " + err.Error()) }

    _, err = writer.Write(output) //save the tree
    if err != nil { halt("writer.Write - " + err.Error()) }
    if err = writer.Sync(); err != nil { halt("writer.Sync - " + err.Error()) }
    if err = writer.Close(); err != nil { halt("writer.Close - " + err.Error()) }
    return
} //end func Export
func Histogram(plotWidth, plotHeight int, pngFile string) {
/*         Purpose : Plots a histogram of the leaf point counts and saves it to the specified PNG file.
 *                   The population mean (mu) and standard deviation (sigma) of the leaf counts are also illustrated.
 *       Arguments : plotWidth  = plot width in pixels
 *                   plotHeight = plot height in pixels
 *                   pngFile    = filename for the resulting PNG histogram plot.
 *         Returns : None.
 * Externals -  In : _stats
 * Externals - Out : None.
 *       Functions : halt
 *         Remarks : Creates a temp file for the histogram data which will be deleted on return.
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if _stats.SIZE == 0  { halt("there's no octree to process") }
    if plotWidth   == 0  { halt("the plot width was not specified") }
    if plotHeight  == 0  { halt("the plot height was not specified") }
    if pngFile     == "" { halt("filename for the PNG histogram plot was not specified") }

    //Write the leaf counts to a temporary file
    refTemp, err := ioutil.TempFile("", "octree_")
    if err != nil { halt("ioutil.TempFile - " + err.Error()) }
    histoData   := filepath.ToSlash(refTemp.Name())
    writer, err := os.Create(histoData)
    if err != nil { halt("os.Create - " + err.Error()) }
    defer writer.Close()
    for _, v := range _stats.LEAFCOUNTS { fmt.Fprintf(writer, "%v\n", v) }
    if err = writer.Sync(); err != nil { halt("writer.Sync - " + err.Error()) }
    if err = writer.Close(); err != nil { halt("writer.Close - " + err.Error()) }
    //Compose the gnuplot commands
    plotCmds := []string{
                 fmt.Sprintf("set terminal pngcairo dashed enhanced size %d,%d", plotWidth, plotHeight),
                 fmt.Sprintf(`set output "%s"`, pngFile),
                 fmt.Sprintf("set xrange [%d:%d]", _stats.XMIN - 1, _stats.XMAX + 1),
                 "set yrange [0:]",
                 "set tics out nomirror",
                 `set grid back lt 0 lw 1 lc rgb "black"`,
                 fmt.Sprintf(`set title "Histogram of %s points into %s leaf nodes.\n(%s, %s points, %s)`,
                             _stats.NUMPTS, _stats.NUMLEAVES, _stats.HOW, _stats.TERMINAL_N, _stats.TIME),
                 fmt.Sprintf(`set xlabel "\nLeaf Point Count\n({/Symbol m}=%s, {/Symbol s}=%s, {/Symbol m}-{/Symbol s}=%s, {/Symbol m}+{/Symbol s}=%s)"`,
                             _stats.MU, _stats.SIGMA,  _stats.LOWER, _stats.UPPER),
                 `set ylabel "Frequency"`,
                 `set style arrow 1 nohead lt 3 lc rgb "blue"`,
                 `set style arrow 2 nohead lt 4 lc rgb "blue"`,
                 `set style arrow 3 filled lt 1 lc rgb "blue"`,
                 `set style arrow 4 nohead lt 1 lc rgb "red" lw 3 front`,
                 //mean - std dev
                 fmt.Sprintf("set arrow as 2 from %s,0 to %s,graph 1", _stats.LOWER, _stats.LOWER),
                 fmt.Sprintf("set arrow as 3 from %s,character 3.5 to %s,0", _stats.LOWER, _stats.LOWER),
                 fmt.Sprintf(`set label "{/Symbol m}-{/Symbol s}" at %s,character 3 center tc rgb "blue"`, _stats.LOWER),
                 //mean
                 fmt.Sprintf("set arrow as 1 from %s,0 to %s,graph 1", _stats.MU, _stats.MU),
                 fmt.Sprintf("set arrow as 3 from %s,character 3.5 to %s,0", _stats.MU, _stats.MU),
                 fmt.Sprintf(`set label "{/Symbol m}" at %s,character 3 center tc rgb "blue"`, _stats.MU ),
                 //mean + std dev
                 fmt.Sprintf("set arrow as 2 from %s,0 to %s,graph 1", _stats.UPPER, _stats.UPPER),
                 fmt.Sprintf("set arrow as 3 from %s,character 3.5 to %s,0", _stats.UPPER, _stats.UPPER),
                 fmt.Sprintf(`set label "{/Symbol m}+{/Symbol s}" at %s,character 3 center tc rgb "blue"`, _stats.UPPER),
                 //warn about any empty leaves
                 "set arrow as 4 from 0,0 to 0," + _stats.NUMEMPTY,
                 //frequency vs count
                 `plot "` + histoData + `" u 1:(1) smooth freq w impulses lw 3 lc rgb "#228B22" notitle`,
                 "quit" }
    //Send the commands to gnuplot
    plotter, err := gnuplot.NewPlotter("", false, false)
    if err != nil { halt("gnuplot.NewPlotter - " + err.Error()) }
    for _, v := range plotCmds { plotter.CheckedCmd("%s", v) }
    plotter.Close()
    //Delete the temp file
    err = os.Remove(histoData)
    for err != nil && strings.Contains(err.Error(), "used by another process") {
        time.Sleep(time.Millisecond)
        err = os.Remove(histoData)
    }
    if err != nil { halt("os.Remove - " + err.Error()) }
} //end func Histogram
func Import(file string) {
/*         Purpose : Imports an octree and its meta data from the specified JSON file.
 *       Arguments : file = data filename.
 *         Returns : None.
 * Externals -  In : node
 * Externals - Out : _octree, _stats
 *       Functions : calcStats, halt
 *         Remarks : None.
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if fi, err := os.Stat(file); (err != nil) || (fi.Size() == 0) {
        halt("the input file cannot be located or is empty")
    }

    var jsonIn jsonOctree

    input, err := ioutil.ReadFile(file) //read the whole file
    if err != nil { halt("ioutil.ReadFile - " + err.Error()) }

    err = json.Unmarshal(input, &jsonIn) // decode the JSON data
    if err != nil { halt("json.Unmarshal - " + err.Error()) }

    _stats.HOW     = jsonIn.HOW
    _stats.SIZE    = jsonIn.SIZE
    _stats.STOP    = jsonIn.STOP
    _stats.TIME    = jsonIn.TIME
    _octree        = make([]node, _stats.SIZE, _stats.SIZE)
    for k, v := range jsonIn.OCTREE {
        _octree[k].N = v.N
        if v.N > _stats.STOP { //parent node
            _octree[k].CENTER, _octree[k].CHILDREN = *(v.CENTER), *(v.CHILDREN)
        } else { //leaf node
            _octree[k].KEYS = v.KEYS
        }
    }
    calcStats()
} //end func Import
func Make(method string, terminal_N int, refPoints *DataSet) {
/*         Purpose : Creates a point-region octree recursively using a specified partitioning method and termination criterion.
 *       Arguments : method     = partitioning method: 'Centroid', 'DataMidPoint', 'Geometric Median' or 'XYZ Medians'.
 *                   terminal_N = termination criterion: maximum number of points in a leaf node (>1).
 *                   refPoints  = reference to the map of float64 data points in R^3, keyed on string identifiers.
 *         Returns : None.
 * Externals -  In : DataSet
 * Externals - Out : _builder, _octree, _stats
 *       Functions : calcStats, halt, makeBuilder
 *         Remarks : The resulting octree is a slice of structures constituting a top-down multi-link node list. Each
 *                   slice element stores a node's characteristics as follows:
 *                   N        => number of data points associated with the node (all nodes),
 *                   CENTER   => array with the partition point coordinates (parent node),
 *                   CHILDREN => array of octree indices for the corresponding child nodes (parent node),
 *                   KEYS     => a CSV string of point identifiers from the given data set (leaf node).
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if !(terminal_N > 1)    { halt(fmt.Sprintf("invalid termination criterion '%d'", terminal_N)) }
    if len(*refPoints) == 0 { halt("there are no points to process") }

    _builder     = makeBuilder(method, terminal_N, len(*refPoints)) //create builder function
    _stats.HOW   = method                                           //record partitioning method
    _stats.STOP  = terminal_N                                       //record termination criterion
    _octree      = nil                                              //clear the octree

    start       := time.Now()                                       //record start of execution
    _builder(refPoints)                                             //build the octree
    _stats.TIME  = time.Since(start)                                //get execution time
    _stats.SIZE  = len(_octree)                                     //get total number of nodes
    calcStats()                                                     //calc various stats
} //end func Make
func Query(refQueryPt *DataCoords) string {
/*         Purpose : Traverses an octree top-down to get the data keys at the leaf node corresponding to the octant in
 *                   which lies the specified query point.
 *       Arguments : refQueryPt = reference to the R^3 coordinates of the query point.
 *         Returns : a CSV string of data-point identifiers.
 * Externals -  In : DataCoords, _octree, _stats
 * Externals - Out : None.
 *       Functions : assignOctant, halt
 *         Remarks : None.
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if _stats.SIZE == 0 { halt("there's no octree to query") }

    var nodeIdx int
    for _octree[nodeIdx].N > _stats.STOP { //while node is a parent node
        nodeIdx = _octree[nodeIdx].CHILDREN[assignOctant(&(_octree[nodeIdx].CENTER),refQueryPt)]
    }
    return _octree[nodeIdx].KEYS
} //end func Query
func Summarize(output ...string) {
/*         Purpose : Outputs the meta data and various statistics regarding the octree to a specified file or Stdout.
 *       Arguments : output[0] = optional output filename; the default is os.Stdout
 *         Returns : None.
 * Externals -  In : _stats
 * Externals - Out : None.
 *       Functions : halt
 *         Remarks : None.
 *         History : v1.0.0 - October 26, 2016 - Original release.
 */
    if _stats.SIZE == 0 { halt("there's no octree to summarize") }

    const summary = `
The octree contains {{.OCTREELEN}} nodes:
 - {{.NUMPARENTS}} are parents,
 - {{.NUMLEAVES}} are leaf nodes of which {{.NUMEMPTY}} are empty.

Partitioning, with the method "{{.HOW}}" and a termination criterion of {{.TERMINAL_N}} points,
resulted in {{.MINPTS}} to {{.MAXPTS}} data points per leaf node
with a mean of {{.MU}} and a population standard deviation of {{.SIGMA}}.

Execution time was {{.TIME}}.

`
    var(
        err    error
        writer *os.File
    )
    switch len(output) { //set output destination
        case 0:
            writer = os.Stdout
        case 1:
            writer, err = os.Create(output[0])
            if err != nil { halt("os.Create - " + err.Error()) }
            defer writer.Close()
        default:
            halt("too many arguments specified")
    }

    err = template.Must(template.New("").Parse(summary)).Execute(writer, _stats)
    if err != nil { halt("executing template - " + err.Error()) }

    if writer != os.Stdout {
        if err = writer.Sync(); err != nil { halt("writer.Sync - " + err.Error()) }
        if err = writer.Close(); err != nil { halt("writer.Close - " + err.Error()) }
    }
    return
} //end func Summarize
//Private ----------------------------------------------------------------------------------------------------------------------
type (
    builderFn func(refPoints *DataSet)                         //octree builder
    centerFn  func(refPoints *DataSet) DataCoords              //center calculator

    jsonNode struct {                                          //JSON structure for an octree node:
        ID          int           `json:"id"`                  // node meta data
        N           int           `json:"N"`                   // node data
        CENTER      *DataCoords   `json:"center,omitempty"`
        CHILDREN    *nodeLinks    `json:"children,omitempty"`
        KEYS        string        `json:"keys,omitempty"`
    }
    jsonOctree struct {                                        //JSON structure for the octree:
        HOW         string        `json:"method"`              // octree meta data
        STOP        int           `json:"terminal_N"`
        TIME        time.Duration `json:"time"`
        SIZE        int           `json:"nodes"`
        OCTREE      []jsonNode    `json:"octree"`              // octree data
    }
    node struct {                                              //octree node structure:
        N           int                                        // number of data points associated with the node
        CENTER      DataCoords                                 // array for the parent's partition point coordinates
        CHILDREN    nodeLinks                                  // array of links for the corresponding child nodes
        KEYS        string                                     // CSV of identifiers for the data points associated with a leaf
    }
    nodeLinks       [8]int                                     //array of links to child nodes
    statistics struct {                                        //octree meta data & statistics:
        //set by funcs Make & Import:
        HOW         string                                     // partitioning method
        SIZE        int                                        // number of octree nodes
        STOP        int                                        // stopping criterion
        TIME        time.Duration                              // execution time
        //set by calcStats
        LEAFCOUNTS  []int                                      // leaf point counts
        MINPTS      string                                     // smallest leaf point count
        MAXPTS      string                                     // largest leaf point count
        NUMPARENTS  string                                     // number of parent nodes
        NUMPTS      string                                     // number of data points
        NUMLEAVES   string                                     // number of leaf nodes
        NUMEMPTY    string                                     // number of leaf nodes with no points
        OCTREELEN   string                                     // number of octree nodes
        TERMINAL_N  string                                     // stopping criterion
        MU          string                                     // mean (mu) leaf point count
        SIGMA       string                                     // population standard deviation (sigma) of leaf point counts
        LOWER       string                                     // mu - sigma
        UPPER       string                                     // mu + sigma
        XMIN        int                                        // smallest leaf point count
        XMAX        int                                        // largest leaf point count
    }
)
const _progressBarLen = 50
var (
    _builder builderFn  //octree builder
    _octree  []node     //octree as a slice of nodes
    _stats   statistics //octree meta data & statistics
)
////Octree build & query
func assignOctant(refCenter, refPoint *DataCoords) (octant int) {
    //Establishes the rule for creating the octree and querying it for the associated points.
    //Returns the assigned octant's index as an integer number in the range [0,7].
    for k := range *refPoint {
        if (*refPoint)[k] > (*refCenter)[k] { octant += (1 << uint(k)) }
    }
    return
} //end func assignOctant
func calcAitkenEstimate(e []DataCoords) (aitken DataCoords) {
    //Estimates the limit of a sequence of numbers in R^3 using Aitken Acceleration.
    for k := range aitken {
        num   := e[1][k] - e[0][k]; num *= num
        denom := e[2][k] - 2.*e[1][k] + e[0][k]
        if denom != 0. { aitken[k] = e[0][k] - num / denom
        } else         { aitken[k] = e[2][k] }
    }
    return
} //end func calcAitkenEstimate
func calcWeiszfeldEstimate(refPoints *DataSet, refEstimate *DataCoords) (weiszfeld DataCoords) {
    //Estimates the value of the geometric median in R^3 using Weiszfeld's fixed-point expression.
    var(
        denom float64
        num   [3]float64
    )
    for _, v := range *refPoints {
        metric := 0. // Euclidean distance of point to estimate
        for k := range num {
            diff := v[k] - (*refEstimate)[k]; metric += diff * diff
        }
        metric  = math.Sqrt(metric)
        denom  += 1./metric
        for k := range num { num[k] += v[k]/metric }
    }
    for k := range num { weiszfeld[k] = num[k] / denom }
    return
} //end func calcWeiszfeldEstimate
func makeBuilder(method string, termination int, numPts int) builderFn {
    var(
        calcCenter    = makeCalcCenter(method) //center calculator
        current       int                      //progress-bar current count
        masterNodeIdx int                      //master node index
        terminal_N    = termination            //termination criterion
        total         = numPts                 //progress-bar total count
    )
    return func(refPoints *DataSet) {
            var(
                childPts [8]DataSet        //data points in child nodes
                numPts   = len(*refPoints) //number of data points
            )
            //Initialize
            _octree                 = append(_octree, node{}) //add blank node to octree
            thisNodeIdx            := masterNodeIdx           //set this node's index value
            _octree[thisNodeIdx].N  = numPts                  //record node's point count
            //Check for a leaf node
            if numPts <= terminal_N {
                i    := 0
                keys := make([]string, numPts, numPts)
                for k := range *refPoints {
                    keys[i] = k
                    i++
                }
                _octree[thisNodeIdx].KEYS  = strings.Join(keys, ",")
                current                   += numPts
                updateProgressBar("octree.Make:", current, total)
                return
            }
            //Compute the partition point
            center := calcCenter(refPoints)
            _octree[thisNodeIdx].CENTER = center
            //Segregate the data points relative to the partition point
            for k := range childPts { childPts[k] = make(DataSet) }
            for k, v := range *refPoints { childPts[assignOctant(&center,&v)][k] = v }
            //Create the child nodes
            for k := range childPts {
                masterNodeIdx++
                _octree[thisNodeIdx].CHILDREN[k] = masterNodeIdx
                _builder(&childPts[k])
            }
           }
} //end func makeBuilder
func makeCalcCenter(method string) centerFn {
    switch method {
        case "Centroid":
            return func(refPoints *DataSet) DataCoords {
                    var(
                        centroid DataCoords
                        numPts   = float64(len(*refPoints))
                    )
                    for _, v := range *refPoints {
                        for k := range centroid { centroid[k] += v[k] }
                    }
                    for k := range centroid { centroid[k] /= numPts }
                    return centroid
                   }
        case "DataMidPoint":
            return func(refPoints *DataSet) DataCoords {
                    type minMax struct {
                        MIN, MAX float64
                    }
                    var dataBounds [3]minMax
                    for k := range dataBounds { //initialize
                        dataBounds[k] = minMax{ math.Inf(1), math.Inf(-1) }
                    }
                    for _, v := range *refPoints { //compute min & max for each axis
                        for k := range dataBounds {
                            dataBounds[k].MIN = math.Min(v[k], dataBounds[k].MIN)
                            dataBounds[k].MAX = math.Max(v[k], dataBounds[k].MAX)
                        }
                    }
                    return DataCoords{ 0.5*(dataBounds[0].MIN + dataBounds[0].MAX),
                                       0.5*(dataBounds[1].MIN + dataBounds[1].MAX),
                                       0.5*(dataBounds[2].MIN + dataBounds[2].MAX) }
                   }
        case "Geometric Median":
            return func(refPoints *DataSet) DataCoords {
                    var(
                        calcCentroid = makeCalcCenter("Centroid")
                        diffs        [4][3]float64 //estimate differences (diff[0] not used)
                        iterations   int           //iteration counter
                        medians      [4]DataCoords //geometric median estimates:
                                                   // [0] : guess
                                                   // [1] : Weiszfeld estimate
                                                   // [2] : Weiszfeld estimate
                                                   // [3] : Aitken estimate
                    )
                    medians[0] = calcCentroid(refPoints) //use centroid as init guess
                    for iterations < MaxIterations {
                        for ctrl := 1; ctrl < 4; ctrl++ {
                            if ctrl != 3 { // calc a new estimate by Picard iteration
                                medians[ctrl] = calcWeiszfeldEstimate(refPoints, &(medians[ctrl-1]))
                            } else {       // calc a new estimate by Aitken extrapolation
                                medians[ctrl] = calcAitkenEstimate(medians[:3])
                            }
                            for k := range diffs[0] { // calc discrepencies between estimates
                                diffs[ctrl][k] = math.Abs(medians[ctrl][k] - medians[ctrl-1][k])
                            }
                            // check for convergence
                            if math.Max(diffs[ctrl][0],math.Max(diffs[ctrl][1],diffs[ctrl][2])) < Tol {
                                return medians[ctrl]
                            }
                        }
                        iterations += 3; // update iteration counter
                        for k := range diffs[0] {
                            if (diffs[1][k] > diffs[2][k]) && (diffs[2][k] > diffs[3][k]) { // check for monotonic trend
                                medians[0][k] = medians[3][k] // update with Aitken value
                            } else {
                                medians[0][k] = medians[2][k] // else with Weiszfeld value
                            }
                        }
                    }
                    halt(fmt.Sprintf("maximum number of iterations (%d) exceeded", MaxIterations))
                    panic("not reached")
                   }
        case "XYZ Medians":
            return func(refPoints *DataSet) DataCoords {
                    var(
                        medians   DataCoords
                        numPts    = len(*refPoints)
                        midIdx    = int(math.Trunc(float64(numPts)/2.))
                        ordinates = make([]float64, numPts)
                    )
                    for k := range medians {
                        i := 0
                        for _, v := range *refPoints {
                            ordinates[i] = v[k]
                            i++
                        }
                        sort.Sort(sort.Float64Slice(ordinates))
                        if numPts % 2 != 0 { medians[k] = ordinates[midIdx]
                        } else             { medians[k] = (ordinates[midIdx-1] + ordinates[midIdx]) / 2. }
                    }
                    return medians
                   }
        default:
            halt("unrecognized method name '" + method + "'")
    }
    panic("not reached")
} //end fun makeCalcCenter
////Reporting
func calcStats() {
    //Compile basic octree statistics
    minPts, maxPts                  := mathutil.MaxInt, mathutil.MinInt
    numParents, numLeaves, numEmpty := 0, 0, 0
    _stats.LEAFCOUNTS                = nil
    for _, v := range _octree {
        if v.N > _stats.STOP {
            numParents++
        } else {
            numLeaves++
            _stats.LEAFCOUNTS = append(_stats.LEAFCOUNTS, v.N)
            minPts, maxPts    = mathutil.Min(minPts, v.N), mathutil.Max(maxPts, v.N)
            if v.N == 0 { numEmpty++ }
        }
    }
    //Compute the population mean (mu) and standard deviation (sigma) of the leaf counts
    mu, sigma := 0., 0.
    for _, v := range _stats.LEAFCOUNTS { mu += float64(v) }
    mu /= float64(numLeaves)
    for _, v := range _stats.LEAFCOUNTS { diff := float64(v) - mu; sigma += diff * diff }
    sigma = math.Sqrt(sigma / float64(numLeaves))
    //Prettify/stringify the results
    _stats.NUMPTS     = humanize.Comma(int64(_octree[0].N))
    _stats.MINPTS     = humanize.Comma(int64(minPts))
    _stats.MAXPTS     = humanize.Comma(int64(maxPts))
    _stats.NUMPARENTS = humanize.Comma(int64(numParents))
    _stats.NUMLEAVES  = humanize.Comma(int64(numLeaves))
    _stats.NUMEMPTY   = humanize.Comma(int64(numEmpty))
    _stats.OCTREELEN  = humanize.Comma(int64(_stats.SIZE))
    _stats.TERMINAL_N = humanize.Comma(int64(_stats.STOP))
    _stats.MU         = fmt.Sprintf("%.2f", mu)
    _stats.SIGMA      = fmt.Sprintf("%.2f", sigma)
    _stats.LOWER      = fmt.Sprintf("%.2f", mu - sigma)
    _stats.UPPER      = fmt.Sprintf("%.2f", mu + sigma)
    _stats.XMIN       = minPts
    _stats.XMAX       = maxPts
} //end func calcStats
func halt(msg string) {
    pc, _, _, ok := runtime.Caller(1)
    details      := runtime.FuncForPC(pc)
    if ok && details != nil {
        log.Fatalln(fmt.Sprintf("\a%s: %s", details.Name(), msg))
    }
    log.Fatalln("\aoctree: FATAL ERROR!")
} //end func halt
func updateProgressBar(title string, current, total int) {
    //code derived from Graham King's post "Pretty command line / console output on Unix in Python and Go Lang"
    //(http://www.darkcoding.net/software/pretty-command-line-console-output-on-unix-in-python-and-go-lang/)
    prefix := fmt.Sprintf("%s: %d / %d ", title, current, total)
    amount := int(0.1 + float32(_progressBarLen) * float32(current) / float32(total))
    remain := _progressBarLen - amount
    bar    := strings.Repeat("\u2588", amount) + strings.Repeat("\u2591", remain)
    os.Stdout.WriteString(prefix + bar + "\r")
    if current == total { os.Stdout.WriteString(strings.Repeat(" ", len(prefix) + _progressBarLen) + "\r") }
    os.Stdout.Sync()
    return
} //end func updateProgressBar
//===== Copyright (c) 2016 Yves Beaudoin - All rights reserved - MIT LICENSE (MIT) - Email: webpraxis@gmail.com ================
//end of Package octree
