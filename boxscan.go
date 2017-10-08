package frame

import (
	"image"

	"github.com/as/frame/box"
)

// bxscan resets the measuring function and calls Bxscan in the embedded run
func (f *Frame) boxscan(s []byte, pt image.Point) (image.Point, image.Point) {
	f.ir.Reset(f.Font)
	f.ir.Boxscan(s, f.maxlines)
	pt = f.wrapMin(pt, &f.ir.Box[0])

	if ForceElasticTabstopExperiment {
		bn := f.ir.Nbox
		for bn != 0 {
			bn = f.ir.Stretch(bn)
		}
	}

	return pt, f.boxscan2D(f.ir, pt)
}

func (f *Frame) boxscan2D(r *box.Run, pt image.Point) image.Point {
	n := 0
	for nb := 0; nb < r.Nbox; nb++ {
		b := &r.Box[nb]
		pt = f.wrapMin(pt, b)
		if pt.Y == f.r.Max.Y {
			r.Nchars -= r.Count(nb)
			r.Delete(nb, r.Nbox-1)
			break
		}
		if b.Nrune > 0 {
			if n = f.fits(pt, b); n == 0 {
				panic("drawRun: fits 0")
			}
			if n != b.Nrune {
				r.Split(nb, n)
				b = &r.Box[nb]
			}
			pt.X += b.Width
		} else {
			if b.BC == '\n' {
				pt = f.wrap(pt)
			} else {
				pt.X += f.plot(pt, b)
			}
		}
	}
	return pt
}
