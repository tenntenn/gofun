package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"time"

	_ "image/png"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/exp/audio"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/sprite"
	"golang.org/x/mobile/exp/sprite/clock"
	"golang.org/x/mobile/exp/sprite/glsprite"
	"golang.org/x/mobile/gl"
)

type State int

const (
	stateStart State = iota
	stateRunning
	stateEnd
)

var (
	startClock = time.Now()
	lastClock  = clock.Time(-1)

	state State = stateStart

	eng   = glsprite.Engine()
	scene *sprite.Node

	player *audio.Player

	texs map[string]sprite.SubTex

	scale float32
	x0    float32
	y0    float32
)

func main() {
	app.Main(func(a app.App) {
		var c event.Config
		for e := range a.Events() {
			switch e := event.Filter(e).(type) {
			case event.Lifecycle:
				switch e.Crosses(event.LifecycleStageVisible) {
				case event.ChangeOn:
					start()
				case event.ChangeOff:
					stop()
				}
			case event.Config:
				//config(e, c)
				c = e
			case event.Draw:
				draw(c)
				a.EndDraw()
			case event.Touch:
				touch(e, c)
			}
		}
	})
}

func start() {
	loadSE()
}

func draw(c event.Config) {
	if scene == nil {
		loadScene(c)
	}

	now := clock.Time(time.Since(startClock) * 60 / time.Second)
	if now == lastClock {
		return
	}
	lastClock = now

	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	eng.Render(scene, now, c)
	//debug.DrawFPS()
}

func stop() {
	player.Close()
}

func touch(t event.Touch, c event.Config) {
	//log.Printf("touch %#v", t)
	if t.ID != 0 && t.Change != event.ChangeOff {
		return
	}

	//log.Println("touch")

	switch state {
	case stateStart:
		state = stateRunning
		startClock = time.Now()
	case stateEnd:
		state = stateStart
	}
}

func newTimeNode() *sprite.Node {
	timeNode := &sprite.Node{}
	eng.Register(timeNode)

	nodes := make([]*sprite.Node, 5)
	for i := range nodes {
		nodes[i] = &sprite.Node{}
		eng.Register(nodes[i])
		eng.SetTransform(nodes[i], f32.Affine{
			{200 * scale, 0, x0 + float32(i)*200*scale},
			{0, 200 * scale, y0},
		})
		timeNode.AppendChild(nodes[i])
	}

	var last time.Duration
	timeNode.Arranger = arrangerFunc(func(eng sprite.Engine, n *sprite.Node, t clock.Time) {
		if state != stateRunning {
			return
		}

		remaining := 60*5 - time.Since(startClock)/time.Second
		if last == remaining {
			return
		}

		if remaining <= 0 {
			//log.Println("GOOOON")
			state = stateEnd
			for i := range nodes {
				eng.SetSubTex(nodes[i], sprite.SubTex{})
			}
			player.Seek(0)
			player.Play()
			return
		}

		now := fmt.Sprintf("%02d:%02d", remaining/60, remaining%60)
		//log.Println(now)
		last = remaining

		for i := range nodes {
			eng.SetSubTex(nodes[i], texs[now[i:i+1]])
		}
	})

	return timeNode
}

func newStartNode() *sprite.Node {
	n := &sprite.Node{}
	eng.Register(n)
	eng.SetTransform(n, f32.Affine{
		{600 * scale, 0, x0 + 200*scale},
		{0, 200 * scale, y0},
	})
	return n
}

func newEndNode() *sprite.Node {
	n := &sprite.Node{}
	eng.Register(n)
	eng.SetTransform(n, f32.Affine{
		{600 * scale, 0, x0 + 200*scale},
		{0, 200 * scale, y0},
	})
	return n
}

func loadScene(c event.Config) {
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	w, h := float32(c.Width), float32(c.Height)

	texs = loadTextures()
	scene = &sprite.Node{}
	eng.Register(scene)
	a := f32.Affine{
		{1, 0, 0},
		{0, 1, 0},
	}

	if h > w {
		w, h = h, w
		angle := float32(-math.Pi / 2)
		a = f32.Affine{
			{f32.Cos(angle), -f32.Sin(angle), 0},
			{f32.Sin(angle), f32.Cos(angle), w},
		}
	}
	scale = w / 1100
	eng.SetTransform(scene, a)
	x0 = w/2 - 500*scale
	y0 = h/2 - 100*scale
	//log.Printf("width:%f height:%f scale:%f x0:%f y0:%f", w, h, scale, x0, y0)

	timeNode := newTimeNode()
	scene.AppendChild(timeNode)

	startNode := newStartNode()
	scene.AppendChild(startNode)

	endNode := newEndNode()
	scene.AppendChild(endNode)

	startVisilbe, endVisible := false, false
	scene.Arranger = arrangerFunc(func(eng sprite.Engine, n *sprite.Node, t clock.Time) {
		switch state {
		case stateStart:
			if !startVisilbe {
				startVisilbe = true
				eng.SetSubTex(startNode, texs["GO"])
			}

			if endVisible {
				eng.SetSubTex(endNode, sprite.SubTex{})
				endVisible = false
			}
		case stateRunning:
			if startVisilbe {
				eng.SetSubTex(startNode, sprite.SubTex{})
				startVisilbe = false
			}
		case stateEnd:
			if !endVisible {
				endVisible = true
				eng.SetSubTex(endNode, texs["gooon"])
			}
		}
	})
}

func loadTextures() map[string]sprite.SubTex {
	a, err := asset.Open("tx_letters.png")
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	img, _, err := image.Decode(a)
	if err != nil {
		log.Fatal(err)
	}
	t, err := eng.LoadTexture(img)
	if err != nil {
		log.Fatal(err)
	}

	return map[string]sprite.SubTex{
		":":     sprite.SubTex{t, image.Rect(0, 0, 200, 200)},
		"0":     sprite.SubTex{t, image.Rect(200, 0, 400, 200)},
		"1":     sprite.SubTex{t, image.Rect(400, 0, 600, 200)},
		"2":     sprite.SubTex{t, image.Rect(0, 200, 200, 400)},
		"3":     sprite.SubTex{t, image.Rect(200, 200, 400, 400)},
		"4":     sprite.SubTex{t, image.Rect(400, 200, 600, 400)},
		"5":     sprite.SubTex{t, image.Rect(0, 400, 200, 600)},
		"6":     sprite.SubTex{t, image.Rect(200, 400, 400, 600)},
		"7":     sprite.SubTex{t, image.Rect(400, 400, 600, 600)},
		"8":     sprite.SubTex{t, image.Rect(0, 600, 200, 800)},
		"9":     sprite.SubTex{t, image.Rect(200, 600, 400, 800)},
		"GO":    sprite.SubTex{t, image.Rect(0, 800, 600, 1000)},
		"gooon": sprite.SubTex{t, image.Rect(0, 1000, 600, 1200)},
	}
}

func loadSE() {
	rc, err := asset.Open("gooon.wav")
	if err != nil {
		log.Fatal(err)
	}
	player, err = audio.NewPlayer(rc, 0, 0)
	if err != nil {
		log.Fatal(err)
	}

	player.SetVolume(1)
	//log.Println("SE open")
}

type arrangerFunc func(e sprite.Engine, n *sprite.Node, t clock.Time)

func (a arrangerFunc) Arrange(e sprite.Engine, n *sprite.Node, t clock.Time) { a(e, n, t) }
