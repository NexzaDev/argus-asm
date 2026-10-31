package gui

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIconAssetsAreWellFormed(t *testing.T) {
	root := filepath.Join("..", "..")
	read := func(rel string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	ico := read("assets/icon/argus-asm.ico")
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[0:2]) != 0 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		t.Fatal("argus-asm.ico is not an ICO file")
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	var sizes []int
	for i := 0; i < count; i++ {
		w := int(ico[6+16*i])
		if w == 0 {
			w = 256
		}
		sizes = append(sizes, w)
	}
	if fmt.Sprint(sizes) != "[16 24 32 48 64 128 256]" {
		t.Fatalf("ico sizes = %v", sizes)
	}

	icns := read("assets/icon/argus-asm.icns")
	if string(icns[:4]) != "icns" || int(binary.BigEndian.Uint32(icns[4:8])) != len(icns) {
		t.Fatal("argus-asm.icns header or length is wrong")
	}
	types := map[string]bool{}
	for i := 8; i+8 <= len(icns); {
		n := int(binary.BigEndian.Uint32(icns[i+4 : i+8]))
		if n < 8 || i+n > len(icns) {
			t.Fatal("argus-asm.icns has a bad chunk length")
		}
		types[string(icns[i:i+4])] = true
		i += n
	}
	for _, want := range []string{"ic07", "ic08", "ic09", "ic10"} {
		if !types[want] {
			t.Fatalf("argus-asm.icns is missing %s", want)
		}
	}

	for _, px := range []int{16, 24, 32, 48, 64, 128, 256, 512, 1024} {
		cfg, err := png.DecodeConfig(bytes.NewReader(read(fmt.Sprintf("assets/icon/png/argus-asm-%d.png", px))))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Width != px || cfg.Height != px {
			t.Fatalf("png %d is %dx%d", px, cfg.Width, cfg.Height)
		}
	}

	for _, rel := range []string{"assets/icon/argus-asm.svg", "assets/icon/argus-asm-16.svg", "assets/icon/linux/argus-asm.svg"} {
		if !strings.Contains(string(read(rel)), "<svg") {
			t.Fatalf("%s is not an SVG", rel)
		}
	}
	if cfg, err := png.DecodeConfig(bytes.NewReader(read("assets/icon/linux/argus-asm.png"))); err != nil || cfg.Width != 512 {
		t.Fatalf("linux png: %v %+v", err, cfg)
	}
}

func TestLinuxDesktopEntryUsesPackagedIcon(t *testing.T) {
	root := filepath.Join("..", "..")
	desktop, err := os.ReadFile(filepath.Join(root, "packaging", "linux", "argus-asm.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desktop), "\nIcon=argus-asm\n") {
		t.Fatal("desktop entry does not reference the icon name argus-asm")
	}
	if _, err := os.Stat(filepath.Join(root, "assets", "icon", "linux", "argus-asm.png")); err != nil {
		t.Fatal(err)
	}
}
