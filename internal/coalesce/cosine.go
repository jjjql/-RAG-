package coalesce

import "math"

// CosineSimilarity 计算 L2 归一化后的余弦相似度（等价于归一化向量点积）。
// 维数不一致或任一向量范数为 0 时返回 ok=false。
func CosineSimilarity(a, b []float64) (sim float64, ok bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var na, nb, dot float64
	for i := range a {
		x, y := a[i], b[i]
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0, false
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb)), true
}
