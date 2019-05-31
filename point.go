package grog

import (
	"fmt"
	"image"
)

type Point struct {
	X float32
	Y float32
}

func PtPt(p image.Point) Point { return Point{float32(p.X), float32(p.Y)} }
func Pt(x, y float32) Point    { return Point{x, y} }
func PtI(x, y int) Point       { return Point{float32(x), float32(y)} }

func (p Point) Add(pt Point) Point  { return Point{p.X + pt.X, p.Y + pt.Y} }
func (p Point) Sub(pt Point) Point  { return Point{p.X - pt.X, p.Y - pt.Y} }
func (p Point) Div(k float32) Point { return Point{p.X / k, p.Y / k} }
func (p Point) Mul(k float32) Point { return Point{p.X * k, p.Y * k} }
func (p Point) Eq(pt Point) bool    { return p.X == pt.X && p.Y == pt.Y }

func (p Point) In(r image.Rectangle) bool {
	return float32(r.Min.X) <= p.X && p.X < float32(r.Max.X) &&
		float32(r.Min.Y) <= p.Y && p.Y < float32(r.Max.Y)
}

func (p Point) String() string {
	return fmt.Sprintf("(%.2f,%.2f)", p.X, p.Y)
}
