// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package draw

import (
	"image"
	"image/color"
	"testing"
)

func eq(c0, c1 color.Color) bool {
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	return r0 == r1 && g0 == g1 && b0 == b1 && a0 == a1
}

func fillBlue(alpha int) image.Image {
	return image.Uniform{color.RGBA{0, 0, uint8(alpha), uint8(alpha)}}
}

func fillAlpha(alpha int) image.Image {
	return image.Uniform{color.Alpha{uint8(alpha)}}
}

func vgradGreen(alpha int) image.Image {
	m := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			m.Set(x, y, color.RGBA{0, uint8(y * alpha / 15), 0, uint8(alpha)})
		}
	}
	return m
}

func vgradAlpha(alpha int) image.Image {
	m := image.NewAlpha(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			m.Set(x, y, color.Alpha{uint8(y * alpha / 15)})
		}
	}
	return m
}

func hgradRed(alpha int) Image {
	m := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			m.Set(x, y, color.RGBA{uint8(x * alpha / 15), 0, 0, uint8(alpha)})
		}
	}
	return m
}

type drawTest struct {
	desc     string
	src      image.Image
	mask     image.Image
	op       Op
	expected color.Color
}

var drawTests = []drawTest{
	// Uniform mask (0% opaque).
	drawTest{"nop", vgradGreen(255), fillAlpha(0), Over, color.RGBA{136, 0, 0, 255}},
	drawTest{"clear", vgradGreen(255), fillAlpha(0), Src, color.RGBA{0, 0, 0, 0}},
	// Uniform mask (100%, 75%, nil) and uniform source.
	// At (x, y) == (8, 8):
	// The destination pixel is {136, 0, 0, 255}.
	// The source pixel is {0, 0, 90, 90}.
	drawTest{"fill", fillBlue(90), fillAlpha(255), Over, color.RGBA{88, 0, 90, 255}},
	drawTest{"fillSrc", fillBlue(90), fillAlpha(255), Src, color.RGBA{0, 0, 90, 90}},
	drawTest{"fillAlpha", fillBlue(90), fillAlpha(192), Over, color.RGBA{100, 0, 68, 255}},
	drawTest{"fillAlphaSrc", fillBlue(90), fillAlpha(192), Src, color.RGBA{0, 0, 68, 68}},
	drawTest{"fillNil", fillBlue(90), nil, Over, color.RGBA{88, 0, 90, 255}},
	drawTest{"fillNilSrc", fillBlue(90), nil, Src, color.RGBA{0, 0, 90, 90}},
	// Uniform mask (100%, 75%, nil) and variable source.
	// At (x, y) == (8, 8):
	// The destination pixel is {136, 0, 0, 255}.
	// The source pixel is {0, 48, 0, 90}.
	drawTest{"copy", vgradGreen(90), fillAlpha(255), Over, color.RGBA{88, 48, 0, 255}},
	drawTest{"copySrc", vgradGreen(90), fillAlpha(255), Src, color.RGBA{0, 48, 0, 90}},
	drawTest{"copyAlpha", vgradGreen(90), fillAlpha(192), Over, color.RGBA{100, 36, 0, 255}},
	drawTest{"copyAlphaSrc", vgradGreen(90), fillAlpha(192), Src, color.RGBA{0, 36, 0, 68}},
	drawTest{"copyNil", vgradGreen(90), nil, Over, color.RGBA{88, 48, 0, 255}},
	drawTest{"copyNilSrc", vgradGreen(90), nil, Src, color.RGBA{0, 48, 0, 90}},
	// Variable mask and variable source.
	// At (x, y) == (8, 8):
	// The destination pixel is {136, 0, 0, 255}.
	// The source pixel is {0, 0, 255, 255}.
	// The mask pixel's alpha is 102, or 40%.
	drawTest{"generic", fillBlue(255), vgradAlpha(192), Over, color.RGBA{81, 0, 102, 255}},
	drawTest{"genericSrc", fillBlue(255), vgradAlpha(192), Src, color.RGBA{0, 0, 102, 102}},
}

func makeGolden(dst image.Image, t drawTest) image.Image {
	// Since golden is a newly allocated image, we don't have to check if the
	// input source and mask images and the output golden image overlap.
	golden := image.NewRGBA(image.Rect(0, 0, dst.Width(), dst.Height()))
	for y := 0; y < golden.Height(); y++ {
		my, sy := y, y
		for x := 0; x < golden.Width(); x++ {
			mx, sx := x, x
			const M = 1<<16 - 1
			var dr, dg, db, da uint32
			if t.op == Over {
				dr, dg, db, da = dst.At(x, y).RGBA()
			}
			sr, sg, sb, sa := t.src.At(sx, sy).RGBA()
			ma := uint32(M)
			if t.mask != nil {
				_, _, _, ma = t.mask.At(mx, my).RGBA()
			}
			a := M - (sa * ma / M)
			golden.Set(x, y, color.RGBA64{
				uint16((dr*a + sr*ma) / M),
				uint16((dg*a + sg*ma) / M),
				uint16((db*a + sb*ma) / M),
				uint16((da*a + sa*ma) / M),
			})
		}
	}
	return golden
}

func TestDraw(t *testing.T) {
loop:
	for _, test := range drawTests {
		dst := hgradRed(255)
		// Draw the (src, mask, op) onto a copy of dst using a slow but obviously correct implementation.
		golden := makeGolden(dst, test)
		// Draw the same combination onto the actual dst using the optimized DrawMask implementation.
		DrawMask(dst, Rect(0, 0, dst.Width(), dst.Height()), test.src, ZP, test.mask, ZP, test.op)
		// Check that the resultant pixel at (8, 8) matches what we expect
		// (the expected value can be verified by hand).
		if !eq(dst.At(8, 8), test.expected) {
			t.Errorf("draw %s: at (8, 8) %v versus %v", test.desc, dst.At(8, 8), test.expected)
			continue
		}
		// Check that the resultant dst image matches the golden output.
		for y := 0; y < golden.Height(); y++ {
			for x := 0; x < golden.Width(); x++ {
				if !eq(dst.At(x, y), golden.At(x, y)) {
					t.Errorf("draw %s: at (%d, %d), %v versus golden %v", test.desc, x, y, dst.At(x, y), golden.At(x, y))
					continue loop
				}
			}
		}
	}
}

// TestIssue836 verifies http://code.google.com/p/go/issues/detail?id=836.
func TestIssue836(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 1, 1))
	b := image.NewRGBA(image.Rect(0, 0, 2, 2))
	b.Set(0, 0, color.RGBA{0, 0, 0, 5})
	b.Set(1, 0, color.RGBA{0, 0, 5, 5})
	b.Set(0, 1, color.RGBA{0, 5, 0, 5})
	b.Set(1, 1, color.RGBA{5, 0, 0, 5})
	Draw(a, Rect(0, 0, 1, 1), b, Pt(1, 1))
	if !eq(color.RGBA{5, 0, 0, 5}, a.At(0, 0)) {
		t.Errorf("Issue 836: want %v got %v", color.RGBA{5, 0, 0, 5}, a.At(0, 0))
	}
}
