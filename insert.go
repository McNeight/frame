package frame

import (
	"image"
	"image/draw"
)

// Put

func (f *Frame) Mark() {
	f.modified = true
}

// Find the points where all the old x and new x line up
// Invariants:
//   pt[0] is where the next box (b, n0) is now
//   pt[1] is where it will be after insertion
// If pt[1] goes out of bounds, we're done
func (f *Frame) alignX(cn0 int64, n0 int, pt0, pt1 image.Point) (int64, int, image.Point, image.Point) {
	type Pts [2]image.Point
	f.pts = f.pts[:0]
	for ; pt1.X != pt0.X && pt1.Y != f.r.Max.Y && n0 < f.Nbox; n0++ {
		b := &f.Box[n0]
		pt0 = f.wrapMax(pt0, b)
		pt1 = f.wrapMin(pt1, b)
		if b.Nrune > 0 {
			if n := f.fits(pt1, b); n != b.Nrune {
				f.Split(n0, n)
				b = &f.Box[n0]
			}
		}
		// check for text overflow off the frame
		if pt1.Y == f.r.Max.Y {
			break
		}
		f.pts = append(f.pts, Pts{pt0, pt1})
		pt0 = f.advance(pt0, b)
		pt1.X += f.plot(pt1, b)
		cn0 += int64(b.Len())
	}
	return cn0, n0, pt0, pt1
}

func (f *Frame) correct(p0 int64, n0, nn0 int, pt0, pt1, ppt1 image.Point) {
	h := f.Font.Dy()
	if n0 == f.Nbox {
		f.Nlines = (pt1.Y - f.r.Min.Y) / h
		if pt1.X > f.r.Min.X {
			f.Nlines++
		}
	} else if pt1.Y != pt0.Y {
		y := f.r.Max.Y
		qt0 := pt0.Y + h
		qt1 := pt1.Y + h
		f.Nlines += (qt1 - qt0) / h
		if f.Nlines > f.maxlines {
			f.trim(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			r := f.r
			r.Min.Y = qt1
			r.Max.Y = y
			if qt1 < y {
				f.Draw(f.b, r, f.b, image.Pt(f.r.Min.X, qt0), f.op)
			}
			r.Min = pt1
			r.Max.X = pt1.X + (f.r.Max.X - pt0.X)
			r.Max.Y = qt1
			f.Draw(f.b, r, f.b, pt0, f.op)
		}
	}
}

func (f *Frame) bltz(cn0 int64, n0 int, pt0, pt1, opt0 image.Point) {
	h := f.Font.Dy()
	y := 0
	if pt1.Y == f.r.Max.Y {
		y = pt1.Y
	}
	x := len(f.pts)
	run := f.Box[n0-x:]
	x--
	_, back := f.pick(cn0, f.p0, f.p1)
	for ; x >= 0; x-- {
		b := &run[x]
		br := image.Rect(0, 0, b.Width, h)
		pt := f.pts[x]
		if b.Nrune > 0 {
			f.Draw(f.b, br.Add(pt[1]), f.b, pt[0], f.op)
			// clear bit hanging off right
			if x == 0 && pt[1].Y > pt0.Y {
				_, back = f.pick(cn0, f.p0, f.p1)
				// line wrap - new char bigger than first char displaced
				r := br.Add(opt0)
				r.Max.X = f.r.Max.X
				f.Draw(f.b, r, back, r.Min, f.op)
			} else if pt[1].Y < y {
				// copy from left to right
				_, back = f.pick(cn0, f.p0, f.p1)

				r := image.ZR.Add(pt[1])
				r.Min.X += b.Width
				r.Max.X += f.r.Max.X
				r.Max.Y += h
				f.Draw(f.b, r, back, r.Min, f.op)
			}
			y = pt[1].Y
			cn0 -= int64(b.Nrune)
		} else {
			r := br.Add(pt[1])
			if r.Max.X >= f.r.Max.X {
				r.Max.X = f.r.Max.X
			}
			cn0--
			_, back = f.pick(cn0, f.p0, f.p1)
			f.Draw(f.b, r, back, r.Min, f.op)
			y = 0
			if pt[1].X == f.r.Min.X {
				y = pt[1].Y
			}
		}
	}
}

type offset struct {
	p0   int64
	n0   int
	cn0  int64
	nn0  int64
	pt0  image.Point
	pt1  image.Point
	opt0 image.Point
}

func (f *Frame) Insert(s []byte, p0 int64) (wrote int) {
	type Pts [2]image.Point
	if p0 > f.Nchars || len(s) == 0 || f.b == nil {
		return
	}

	// find p0, it's box, and its point in the box its in
	n0 := f.Find(0, 0, p0)
	on0 := n0
	cn0 := p0
	nn0 := n0
	pt0 := f.ptOfCharNBox(p0, n0)
	opt0 := pt0

	// find p1
	ppt0, pt1 := f.bxscan(s, pt0)
	ppt1 := pt1

	// Line wrap
	if n0 < f.Nbox {
		b := &f.Box[n0]
		pt0 = f.wrapMax(pt0, b)
		ppt1 = f.wrapMin(ppt1, b)
	}
	f.modified = true
	if f.p0 == f.p1 {
		f.tickat(f.PointOf(int64(f.p0)), false)
	}

	cn0, n0, pt0, pt1 = f.alignX(cn0, n0, pt0, pt1)

	if pt1.Y == f.r.Max.Y && n0 < f.Nbox {
		f.Nchars -= f.Count(n0)
		f.Run.Delete(n0, f.Nbox-1)
	}

	f.correct(p0, n0, nn0, pt0, pt1, ppt1)
	f.bltz(cn0, n0, pt0, pt1, opt0)

	text, back := f.pick(p0, f.p0+1, f.p1+1)
	f.Paint(ppt0, ppt1, back)
	f.redrawRun0(f.ir, ppt0, text, back)
	f.Add(nn0, f.ir.Nbox)
	for n := 0; n < f.ir.Nbox; n++ {
		f.Box[nn0+n] = f.ir.Box[n]
	}
	if nn0 > 0 && f.Box[nn0-1].Nrune >= 0 && ppt0.X-f.Box[nn0-1].Width >= f.r.Min.X {
		nn0--
		ppt0.X -= f.Box[nn0].Width
	}

	n0 += f.ir.Nbox
	if n0 < f.Nbox-1 {
		n0++
	}
	f.clean(ppt0, nn0, n0)
	f.Nchars += f.ir.Nchars
	f.p0, f.p1 = coInsert(p0, p0+f.Nchars, f.p0, f.p1)
	if f.p0 == f.p1 {
		f.tickat(f.PointOf(f.p0), true)
	}
	if ForceElasticTabstopExperiment {
		// Just to see if the algorithm works not ideal to sift through all of
		// the boxes per insertion, although surprisingly faster than expected
		// to the point of where its almost unnoticable without the print
		// statements
		f.Stretch(on0)
		f.Refresh() // must do this until line mapper is fixed
	}
	return int(f.ir.Nchars)
}

func (f *Frame) pick(c, p0, p1 int64) (text, back image.Image) {
	if p0 <= c && c < p1 {
		return f.Color.Hi.Text, f.Color.Hi.Back
	}
	return f.Color.Text, f.Color.Back
}

func region(c, p0, p1 int64) int {
	if c < p0 {
		return -1
	}
	if c >= p1 {
		return 1
	}
	return 0
}

func drawBorder(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, thick int) {
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thick), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-thick, r.Max.X, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thick, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Max.X-thick, r.Min.Y, r.Max.X, r.Max.Y), src, sp, draw.Src)
}
