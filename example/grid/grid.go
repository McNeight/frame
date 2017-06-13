package main

import (
	"bufio"
	"fmt"
	"image"
	"os"

	"github.com/as/frame"
	"github.com/as/frame/tag"
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

func mkfont(size int) frame.Font {
	return frame.NewTTF(gomono.TTF, size)
}

// Put
var (
	winSize = image.Pt(1900, 1000)
	pad     = image.Pt(25, 5)
	fsize   = 11
)

type Plane interface {
	Loc() image.Rectangle
	Move(image.Point)
	Resize(image.Point)
}

// Put
func active(e mouse.Event, act Plane, list ...Plane) (x Plane) {
	if tag.Buttonsdown != 0 {
		return act
	}
	pt := image.Pt(int(e.X), int(e.Y))
	if act != nil {
		pt = pt.Add(act.Loc().Min)
		list = append([]Plane{act}, list...)
	}
	for i, w := range list {
		r := w.Loc()
		if pt.In(r) {
			return list[i]
		}
	}
	return act
}

type Col struct {
	sp   image.Point
	size image.Point
	wind screen.Window
	List []Plane
}

var cols = frame.Acme

func NewCol(src screen.Screen, wind screen.Window, ft frame.Font, sp, size image.Point, files ...string) *Col {
	N := len(files)
	dy := size.Y / N
	n := 0
	col := &Col{sp: sp, size: size, wind: wind, List: make([]Plane, len(files))}
	for i, v := range files {
		sp = image.Pt(sp.X, n*dy)
		dp := image.Pt(size.X, dy)
		n++
		fmt.Printf("sp=%s size=%s\n", sp, dp)
		t := tag.NewTag(src, wind, ft, sp, dp, pad, cols)
		t.Open(v)
		col.List[i] = t
	}
	return col
}

func (co *Col) Upload(wind screen.Window) {
	for _, t := range co.List {
		t.(*tag.Tag).Upload(wind)
	}
}

func (co *Col) Resize(size image.Point) {
	co.size = size
	N := len(co.List)
	dy := size.Y / N
	sp := image.Pt(co.sp.X, co.sp.Y)
	dp := image.Pt(size.X, dy)
	for _, t := range co.List {
		t.Move(sp)
		t.Resize(dp)
		sp = sp.Add(image.Pt(0, dy))
	}
}

// Put
func main() {
	driver.Main(func(src screen.Screen) {
		wind, _ := src.NewWindow(&screen.NewWindowOptions{winSize.X, winSize.Y})
		wind.Send(paint.Event{})
		focused := false
		focused = focused
		ft := mkfont(fsize)

		list := []string{"/dev/stdin"}
		if len(os.Args) > 1 {
			list = os.Args[1:]
		}
		co := NewCol(src, wind, ft, image.ZP, winSize, list...)
		act := co.List[0].(*tag.Tag).W
		actTag := co.List[0]

		go func() {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				if x := sc.Text(); x == "u" || x == "r" {
					act.SendFirst(x)
					continue
				}
				act.SendFirst(tag.Cmdparse(sc.Text()))
			}
		}()
		for {
			// Put
			switch e := act.NextEvent().(type) {
			case mouse.Event:
				actTag = active(e, actTag, co.List...).(*tag.Tag)
				act = active(e, act, actTag.(*tag.Tag).W, actTag.(*tag.Tag).Wtag).(*tag.Invertable)
				actTag.(*tag.Tag).Handle(act, e)
			case string, *tag.Command, tag.ScrollEvent, key.Event:
				actTag.(*tag.Tag).Handle(act, e)
			case size.Event:
				winSize = image.Pt(e.WidthPx, e.HeightPx)
				co.Resize(winSize)
				act.SendFirst(paint.Event{})
			case paint.Event:
				co.Upload(wind)
				wind.Publish()
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
				// NT doesn't repaint the window if another window covers it
				if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOff {
					focused = false
				} else if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOn {
					focused = true
				}
			}
		}
	})

}
