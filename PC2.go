package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Data struct {
	Attrs []string
	Label string
}

type TreeNode struct {
	AttributeIndex int
	Class          string
	Children       map[string]*TreeNode
	IsLeaf         bool
}

func gini(data []Data) float64 {
	m := make(map[string]int)
	for _, d := range data {
		m[d.Label]++
	}
	n := len(data)
	var g float64
	for _, c := range m {
		p := float64(c) / float64(n)
		g += p * (1 - p)
	}
	return g
}

func split(data []Data, index int) map[string][]Data {

	s := make(map[string][]Data)

	for _, d := range data {
		key := d.Attrs[index]
		s[key] = append(s[key], d)
	}

	return s
}

func bestSplit(data []Data) int {
	best := -1
	bestGini := 1.0

	for i := range data[0].Attrs {
		s := split(data, i)
		var wg float64

		for _, subset := range s {
			p := float64(len(subset)) / float64(len(data))
			wg += p * gini(subset)
		}

		if wg < bestGini {
			bestGini = wg
			best = i
		}
	}

	return best
}

func allSameLabel(data []Data) bool {

	if len(data) == 0 {
		return true
	}

	ref := data[0].Label

	for _, d := range data {
		if d.Label != ref {
			return false
		}
	}

	return true
}

func mostCommonLabel(data []Data) string {
	m := make(map[string]int)

	for _, d := range data {
		m[d.Label]++
	}

	var max int
	var label string

	for k, v := range m {
		if v > max {
			max = v
			label = k
		}
	}

	return label
}

func buildTree(data []Data) *TreeNode {

	if len(data) == 0 || allSameLabel(data) {
		return &TreeNode{
			Class:  mostCommonLabel(data),
			IsLeaf: true,
		}

	}

	best := bestSplit(data)
	s := split(data, best)
	c := make(map[string]*TreeNode)

	for k, sub := range s {
		c[k] = buildTree(sub)
	}

	return &TreeNode{
		AttributeIndex: best,
		Children:       c,
		IsLeaf:         false,
	}
}

func buildTreeConcurrent(data []Data) *TreeNode {

	if len(data) == 0 || allSameLabel(data) {
		return &TreeNode{
			Class:  mostCommonLabel(data),
			IsLeaf: true,
		}
	}

	best := bestSplit(data)
	s := split(data, best)
	c := make(map[string]*TreeNode)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for k, sub := range s {
		wg.Add(1)
		go func(k string, sub []Data) {
			defer wg.Done()
			node := buildTreeConcurrent(sub)
			mu.Lock()
			c[k] = node
			mu.Unlock()
		}(k, sub)
	}

	wg.Wait()

	return &TreeNode{
		AttributeIndex: best,
		Children:       c,
		IsLeaf:         false,
	}
}

func predict(n *TreeNode, input []string) string {

	if n.IsLeaf {
		return n.Class
	}

	v := input[n.AttributeIndex]
	child, ok := n.Children[v]

	if !ok {
		return n.Class
	}

	return predict(child, input)
}

func loadCSV(path string) ([]Data, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	var out []Data
	_, _ = r.Read()
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, Data{
			Attrs: row[1:],
			Label: row[0],
		})
	}
	return out, nil
}

func main() {
	data, err := loadCSV("mushrooms.csv")
	if err != nil {
		panic(err)
	}

	t1 := time.Now()
	treeSeq := buildTree(data)
	t2 := time.Since(t1)
	fmt.Println("Algoritmo sequencial:", t2)

	t3 := time.Now()
	treeConc := buildTreeConcurrent(data)
	t4 := time.Since(t3)
	fmt.Println("Algoritmo concurrente:", t4)

	fmt.Println("Predicción secuencial:", predict(treeSeq, data[0].Attrs))
	fmt.Println("Predicción concurrente:", predict(treeConc, data[0].Attrs))
}
