package consume2_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/keep94/consume2"
	"github.com/stretchr/testify/assert"
)

func TestPipeline(t *testing.T) {
	pipeline := consume2.PFilter(func(x int) bool { return x%2 == 1 })
	pipeline = consume2.Join(
		pipeline, consume2.PMap(func(x int) int { return 3 * x }))
	pipelineStr := consume2.Join(
		pipeline, consume2.PMap(strconv.Itoa))
	pipelineStr = consume2.Join(
		pipelineStr, consume2.PSlice[string](3, 8))
	var x []string
	feedInts(pipelineStr.AppendTo(&x))
	assert.Equal(t, []string{"21", "27", "33", "39", "45"}, x)
}

func TestPipeline2(t *testing.T) {
	people := []person{
		{Name: "a", Age: 1},
		{Name: "b", Age: 2},
		{Name: "c", Age: 3},
		{Name: "d", Age: 4},
		{Name: "e", Age: 5},
		{Name: "f", Age: 6},
		{Name: "g", Age: 1},
		{Name: "h", Age: 2},
		{Name: "i", Age: 3},
		{Name: "j", Age: 4},
		{Name: "k", Age: 5},
	}
	pipeline := consume2.PTakeWhile(func(p person) bool { return p.Age < 6 })
	pipeline = consume2.Join(
		pipeline,
		consume2.PFilterp(func(p *person) bool {
			p.Name = strings.ToUpper(p.Name)
			return true
		}))
	pipelineStr := consume2.Join(
		pipeline,
		consume2.PMaybeMap(func(p person) (string, bool) {
			if p.Age%2 == 0 {
				return "", false
			}
			return p.Name, true
		}))
	var answer []string
	consume2.FromSlice(people, pipelineStr.AppendTo(&answer))
	assert.Equal(t, []string{"A", "C", "E"}, answer)
}
