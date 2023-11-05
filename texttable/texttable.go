package texttable

import "math"

func printArrayColumns(list []string, columns int) string {
	columnLengths := []int{}

	perColumn := int(math.Ceil(float64(len(list)) / float64(columns)))

	for i := 0; i < columns; i++ {
		maxLen := 0
		for row := 0; row < perColumn; row++ {
			elem := i*perColumn + row
			if elem < len(list) {
				if len(list[elem]) > maxLen {
					maxLen = len(list[elem])
				}
			}
		}
		columnLengths = append(columnLengths, maxLen)
	}

	result := ""
	for row := 0; row < perColumn; row++ {
		for column := 0; column < columns; column++ {
			value := list[column*perColumn+row]
			// Don't put added spaces on the last column.
			if column != columns-1 {
				for len(value) < columnLengths[column] {
					value += " "
				}
				// Always add one space to separate the columns.
				value += " "
			}
			result += value
		}
		result += "\n"
	}

	return result
}
