package utils

import (
	"math"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
)

var (
	defaultPage     = int32(1)
	defaultPageSize = int32(10)
)

func NewPage(page, pageSize int32) *v1alpha1.Pagination {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	return &v1alpha1.Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    0,
		Pages:    0,
	}
}

func SetPageTotal(p *v1alpha1.Pagination, total int32) *v1alpha1.Pagination {
	p.Total = total
	p.Pages = int32(math.Ceil(float64(total) / float64(p.PageSize)))
	return p
}
