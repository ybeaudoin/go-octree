# octree

Go package for creating, reporting, importing, exporting and querying point-region octrees..

## Install

To install the package and its demo:
```sh
go get -u github.com/ybeaudoin/go-xyztree
```

The package imports binet's gnuplot, cznic's mathutil and dustin's go-humanize packages which can be installed as follows:
```sh
go get -u "bitbucket.org/binet/go-gnuplot/pkg/gnuplot" "github.com/cznic/mathutil" "github.com/dustin/go-humanize"
```
The first package requires that a gnuplot executable be installed and be findable via the environment path statement.
See http://www.gnuplot.info/download.html for available versions.

## At a glance

The package exports the following:
 * Types:
   * `DataCoords`  
     Array for a data point's float64 R<sup>3</sup> coordinates, i.e., \[1.,2.,3.\]
   * `DataSet`  
     Map for the R<sup>3</sup> data points keyed on string identifiers, i.e., "Pt1":\[1.,2.,3.\], "Pt2":\[4.,5.,6.\], etc.
 * Variables:
   * `MaxIterations`  
     Maxinum number of iterations for the geometric-median partition method: default is 1,000.
   * `Tol`  
     Convergence tolerance for the geometric-median partition method: defaul is 1.0E-12.
 * Functions:
   * `Export(file string, compact bool)`  
     Exports the octree and its meta data to a specified file using the JSON format with or without newlines and identations.
   * `Histogram(plotWidth, plotHeight int, pngFile string)`  
     Plots a histogram of the leaf point counts and saves it to the specified PNG file.
     The population mean (mu) and standard deviation (sigma) of the leaf counts are also illustrated.
   * `Import(file string)`  
     Imports an octree and its meta data from the specified JSON file.
   * `Make(method string, terminal_N int, refPoints *DataSet)`  
     Creates a point-region octree recursively using a specified partitioning method and termination criterion.
   * `Query(refQueryPt *DataCoords) string`  
     Traverses an octree top-down to get the data keys at the leaf node corresponding to the octant in
     which lies the specified query point.
   * `Summarize(output ...string)`
     Outputs the meta data and various statistics regarding the octree to a specified file or Stdout.

## Octree

The octree is stored as a slice of structures constituting a top-down multi-link node list. Each slice element contains a node's
particulars as follows:

| Field | Description |
| --- | --- |
|N|number of data points associated with the node (all nodes)|
|CENTER|array with the partition point coordinates (parent node)|
|CHILDREN|array of octree indices for the corresponding child nodes (parent node)|
|KEYS|a CSV string of point identifiers from the given data set (leaf node)|

## Partitioning Methods

There are numerous partioning schemes possible<sup>[\[1\]](https://en.wikipedia.org/wiki/Octree)</sup>. This package offers four
recursive methods called **Centroid**, **DataMidPoint**, **Geometric Median** and **XYZ Medians**. The common termination
criterion is that the number of points in a leaf node be less than or equal to a given value.

* The **Centroid** method takes "the mean position of all the points in all of the coordinate directions"
  <sup>[\[2\]](https://en.wikipedia.org/wiki/Centroid)</sup>.
* The **DataMidPoint** method splits the data at the midpoint between the coordinate minimum and maximum values.
* The **XYZ Medians** scheme partitions using the ordinate medians if the number of points is odd, otherwise it uses an average
  of the two central points<sup>[\[3\]](https://en.wikipedia.org/wiki/Median)</sup>.
* The **Geometric Median** method splits at "the point minimizing the sum of \[Euclidean\] distances to the sample points"
  <sup>[\[4\]](https://en.wikipedia.org/wiki/Geometric_median)</sup>. This point is commonly estimated using *Weiszfeld's
  algorithm*<sup>[\[4\]](https://en.wikipedia.org/wiki/Geometric_median)</sup>:  
  ![](https://wikimedia.org/api/rest_v1/media/math/render/svg/b3fb215363358f12687100710caff0e86cd9d26b)  
  Although the iterative method "converges for almost all initial positions"
  <sup>[\[4\]](https://en.wikipedia.org/wiki/Geometric_median)</sup>, it is slow being computationally expensive. One
  can increase its rate of convergence by first observing that, for each ordinate, the above equation is a fixed-point iteration
  with a linear rate of convergence. To increase the rate, *Aitken extrapolation*
  <sup>[\[5\]](https://en.wikipedia.org/wiki/Aitken%27s_delta-squared_process)</sup> can be used for example. When the formula
  for the extrapolated value is fed back into a fixed-point formula, one obtains *Steffensen's method*
  <sup>[\[6\]](https://en.wikipedia.org/wiki/Steffensen%27s_method)</sup>. Though its rate of convergence is quadratic, it is
  not robust in that it is at times prone to oscillations and even divergence if the starting value is not close enough. Having
  encountered this behavior with one of our data sets, we have devised a cautionary approach. If the successive absolute
  differences between an initial guess, two successive Weiszfeld estimates and an Aitken estimate are monotonically decreasing,
  then the Aitken value is used as a new initial guess to Weiszfeld's algorithm and the process is repeated. Otherwise, it is
  rejected and the Weiszfeld's algorithm repeats with its most current value as a new guess. This approach is applied to each
  ordinate separately with convergence to a given tolerance checked at each stage. In our tests, the convergence rate is between
  linear and quadratic, resulting in better execution times.

## References

1. https://en.wikipedia.org/wiki/Octree
2. https://en.wikipedia.org/wiki/Centroid
3. https://en.wikipedia.org/wiki/Median
4. https://en.wikipedia.org/wiki/Geometric_median
5. https://en.wikipedia.org/wiki/Aitken%27s_delta-squared_process
6. https://en.wikipedia.org/wiki/Steffensen%27s_method

## MIT License

Copyright (c) 2016 Yves Beaudoin webpraxis@gmail.com

See the file LICENSE for copying permission.

















