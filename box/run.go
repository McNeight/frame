package box

import (
	"fmt"
	"github.com/as/frame/font"
)

// MaxBytes is the largest capacity of bytes in a box
var MaxBytes = 512 + 3

func NewRun(minDx, maxDx int, ft *font.Font, newRulerFunc ...func([]byte, *font.Font) Ruler) Run {
	fn := NewByteRuler
	if len(newRulerFunc) > 0 {
		fn = newRulerFunc[0]
	}
	return Run{
		delta:        32,
		minDx:        minDx,
		maxDx:        maxDx,
		Font:         ft,
		newRulerFunc: fn,
		br:           fn(make([]byte, MaxBytes), ft),
	}
}

// Run is a one-dimensional field of boxes. It can scan arbitrary text
// into boxes with Bxscan().
type Run struct {
	*font.Font
	Nchars int64
	Nlines int
	Nalloc int
	Nbox   int
	Box    []Box

	minDx, maxDx int
	delta        int

	newRulerFunc func([]byte, *font.Font) Ruler
	br           Ruler
}

func (f *Run) Combine(g *Run, n int) {
	b := g.Box[:g.Nbox]
	f.Add(n, len(b))
	copy(f.Box[n:], b)
}

// Count recomputes and returns the number of bytes
// stored between box nb and the last box
func (f *Run) Count(nb int) int64 {
	n := int64(0)
	for ; nb < f.Nbox; nb++ {
		n += int64((f.Box[nb]).Len())
	}
	return n
}

// Reset resets all boxes in the run without deallocating
// their data on the heap. If widthfn is not nill, it
// becomes the new measuring function for the run. Boxes
// in the run are not remeasured upon reset.
func (f *Run) Reset(ft *font.Font) {
	f.Nbox = 0
	f.Nchars = 0
	if ft != nil {
		f.Font = ft
	}
}

//Find finds the box containing q starting from box bn index
// p and puts q at the start of the next box
func (f *Run) Find(bn int, p, q int64) int {
	//	fmt.Printf("find %d.%d -> %d\n",bn,p,q)
	for ; bn < f.Nbox; bn++ {
		b := &f.Box[bn]
		if p+int64(b.Len()) > q {
			break
		}
		p += int64(b.Len())
	}
	if p != q {
		f.Split(bn, int(q-p))
		bn++
	}
	//	fmt.Printf("find %d.%d -> %d = box %d\n",bn,p,q, bn)
	return bn
}

func (f *Run) DumpBoxes() {
	return
	fmt.Println("dumping boxes")
	fmt.Printf("nboxes: %d\n", f.Nbox)
	fmt.Printf("nalloc: %d\n", f.Nalloc)
	for i, b := range f.Box {
		fmt.Printf("[%d] (%p) (nrune=%d l=%d w=%d mw=%d bc=%x): %q\n",
			i, &f.Box[i], b.Nrune, (&b).Len(), b.Width, b.Minwidth, b.Break(), b.Ptr)
	}
}

// Merge merges box bn and bn+1
func (f *Run) Merge(bn int) {
	b0 := &f.Box[bn]
	b1 := &f.Box[bn+1]
	b0.Ptr = append(b0.Ptr, b1.Ptr...)
	b0.Width += b1.Width
	b0.Nrune += b1.Nrune
	f.Delete(bn+1, bn+1)
}

// Split splits box bn into two boxes; bn and bn+1, at index n
func (f *Run) Split(bn, n int) {
	f.Dup(bn)
	f.Truncate(&f.Box[bn], (&f.Box[bn]).Nrune-n)
	f.Chop(&f.Box[bn+1], n)
}

func (f *Run) MeasureBytes(p []byte) int {
	f.br.Reset(p)
	for {
		_, _, err := f.br.Next()
		if err != nil {
			break
		}
	}
	return f.br.Width()
}

// Chop drops the first n chars in box b
func (f *Run) Chop(b *Box, n int) {
	if b.Nrune < 0 || b.Nrune < n {
		panic("Chop")
	}
	copy(b.Ptr, b.Ptr[n:])
	b.Nrune -= n
	b.Ptr = b.Ptr[:b.Nrune]
	b.Width = f.MeasureBytes(b.Ptr)
}

func (f *Run) Truncate(b *Box, n int) {
	if b.Nrune < 0 || b.Nrune < n {
		panic("Truncate")
	}
	b.Nrune -= n
	b.Ptr = b.Ptr[:b.Nrune]
	b.Width = f.MeasureBytes(b.Ptr)
}

// Add adds n boxes after box bn, the rest are shifted up
func (f *Run) Add(bn, n int) {
	if bn > f.Nbox {
		panic("Frame.Add")
	}
	if f.Nbox+n > f.Nalloc {
		f.Grow(n + SLOP)
	}
	for i := f.Nbox - 1; i >= bn; i-- {
		f.Box[i+n] = f.Box[i]
	}
	f.Nbox += n
}

// Delete closes and deallocates n0-n1 inclusively
func (f *Run) Delete(n0, n1 int) {
	if n0 >= f.Nbox || n1 >= f.Nbox || n1 < n0 {
		panic("Delete")
	}
	f.Free(n0, n1)
	f.Close(n0, n1)
}

// Free deallocates memory for boxes n0-n1 inclusively
func (f *Run) Free(n0, n1 int) {
	if n1 < n0 {
		return
	}
	if n0 >= f.Nbox || n1 >= f.Nbox {
		panic("Free")
	}
	for i := n0; i < n1; i++ {
		if f.Box[i].Nrune >= 0 {
			f.Box[i].Ptr = nil
		}
	}
}

// Grow allocates memory for delta more boxes
func (f *Run) Grow(delta int) {
	f.Nalloc += delta
	f.Box = append(f.Box, make([]Box, delta)...)
}

// Dup copies the contents of box bn to box bn+1
func (f *Run) Dup(bn int) {
	if f.Box[bn].Nrune < 0 {
		panic("Frame.Dup")
	}
	f.Add(bn, 1)
	if f.Box[bn].Nrune >= 0 {
		f.Box[bn+1].Ptr = append([]byte{}, f.Box[bn].Ptr...)
	}
}

// Close closess box n0-n1 inclusively. The rest are shifted down
func (f *Run) Close(n0, n1 int) {
	if n0 >= f.Nbox || n1 >= f.Nbox || n1 < n0 {
		panic("Frame.Close")
	}
	n1++
	for i := n1; i < f.Nbox; i++ {
		f.Box[i-(n1-n0)] = f.Box[i]
	}
	f.Nbox -= n1 - n0
}

func (b Run) String() string {
	s := ""
	bn, Nbox := 0, b.Nbox
	for ; bn < Nbox; bn++ {
		b := &b.Box[bn]
		s += string(b.Ptr)
	}
	return s
}
