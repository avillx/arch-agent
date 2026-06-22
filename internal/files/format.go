package files

import "fmt"

func FormatSize(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fmb", float64(n)/1024/1024)
	case n >= 1024:
		return fmt.Sprintf("%.1fkb", float64(n)/1024)
	default:
		return fmt.Sprintf("%db", n)
	}
}