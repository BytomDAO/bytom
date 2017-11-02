// Package matrix implements a simple library for creating and
// manipulating matrices, and performing basic linear algebra.

package matrix

import (
	"fmt"
	"strconv"
)

type Matrix struct {
	rows, columns int    // the number of rows and columns.
	data          []int8 // the contents of the matrix as one long slice.
}

// Set lets you define the value of a matrix at the given row and
// column.

func (A *Matrix) Set(r int, c int, val int8) {
	A.data[findIndex(r, c, A)] = val
}

// Get retrieves the contents of the matrix at the row and column.

func (A *Matrix) Get(r, c int) int8 {
	return A.data[findIndex(r, c, A)]
}

// Print converts the matrix into a string and then outputs it to fmt.Printf.

func (A *Matrix) Print() {

	// Find the width (in characters) that each column needs to be.  We hold these
	// widths as strings, not ints, because we're going to use these in a printf
	// function.

	columnWidths := make([]string, A.columns)

	for i := range columnWidths {
		var maxLength int
		thisColumn := A.Column(i + 1)
		for j := range thisColumn {
			thisLength := len(strconv.Itoa(int(thisColumn[j])))
			if thisLength > maxLength {
				maxLength = thisLength
			}
		}
		columnWidths[i] = strconv.Itoa(maxLength)
	}

	// We have the widths, so now output each element with the correct column
	// width so that they line up properly.

	for i := 0; i < A.rows; i++ {
		thisRow := A.Row(i + 1)
		fmt.Printf("[")
		for j := range thisRow {
			var printFormat string
			if j == 0 {
				printFormat = "%" + columnWidths[j] + "s"
			} else {
				printFormat = " %" + columnWidths[j] + "s"
			}
			fmt.Printf(printFormat, strconv.Itoa(int(thisRow[j])))
		}
		fmt.Printf("]\n")
	}
}

// Column returns a slice that represents a column from the matrix.
// This works by examining each row, and adding the nth element of
// each to the column slice.

func (A *Matrix) Column(n int) []int8 {
	col := make([]int8, A.rows)
	for i := 1; i <= A.rows; i++ {
		col[i-1] = A.Row(i)[n-1]
	}
	return col
}

// Row returns a slice that represents a row from the matrix.

func (A *Matrix) Row(n int) []int8 {
	return A.data[findIndex(n, 1, A):findIndex(n, A.columns+1, A)]
}

// Multiply multiplies two matrices together and return the resulting matrix.
// For each element of the result matrix, we get the dot product of the
// corresponding row from matrix A and column from matrix B.

func Multiply(A, B Matrix) *Matrix {
	C := Zeros(A.rows, B.columns)
	for r := 1; r <= C.rows; r++ {
		A_row := A.Row(r)
		for c := 1; c <= C.columns; c++ {
			B_col := B.Column(c)
			C.Set(r, c, dotProduct(A_row, B_col))
		}
	}
	return &C
}

// Add adds two matrices together and returns the resulting matrix.  To do
// this, we just add together the corresponding elements from each matrix.

func Add(A, B Matrix) Matrix {
	C := Zeros(A.rows, A.columns)
	for r := 1; r <= A.rows; r++ {
		for c := 1; c <= A.columns; c++ {
			C.Set(r, c, A.Get(r, c)+B.Get(r, c))
		}
	}
	return C
}

// Identity creates an identity matrix with n rows and n columns.  When you
// multiply any matrix by its corresponding identity matrix, you get the
// original matrix.  The identity matrix looks like a zero-filled matrix with
// a diagonal line of one's starting at the upper left.

func Identity(n int) Matrix {
	A := Zeros(n, n)
	for i := 0; i < len(A.data); i += (n + 1) {
		A.data[i] = 1
	}
	return A
}

// Zeros creates an r x c sized matrix that's filled with zeros.  The initial
// state of an int is 0, so we don't have to do any initialization.

func Zeros(r, c int) Matrix {
	return Matrix{r, c, make([]int8, r*c)}
}

// New creates an r x c sized matrix that is filled with the provided data.
// The matrix data is represented as one long slice.

func New(r, c int, data []int8) Matrix {
	if len(data) != r*c {
		panic("[]int data provided to matrix.New is great than the provided capacity of the matrix!'")
	}
	A := Zeros(r, c)
	A.data = data
	return A
}

// findIndex takes a row and column and returns the corresponding index
// from the underlying data slice.

func findIndex(r, c int, A *Matrix) int {
	return (r-1)*A.columns + (c - 1)
}

// dotProduct calculates the algebraic dot product of two slices.  This is just
// the sum  of the products of corresponding elements in the slices.  We use
// this when we multiply matrices together.

func dotProduct(a, b []int8) int8 {
	var total int32
	for i := 0; i < len(a); i++ {
		total += int32(a[i]) * int32(b[i])
	}
	return int8((total & 0xff) +
		((total >> 8) & 0xff) +
		((total >> 16) & 0xff) +
		((total >> 24) & 0xff))
}
